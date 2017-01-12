// Hash commands such as #flip, dice and #8ball

package parser

import (
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
)

var (
	diceRegexp = regexp.MustCompile(`(\d*)d(\d+)`)

	errTooManyRolls = diceError(0)
	errDieTooBig    = diceError(1)
)

type diceError int

func (d diceError) Error() string {
	return "dice error: " + strconv.Itoa(int(d))
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Parse a matched hash command
func parseCommand(match, board string) (common.Command, error) {

	// TODO: #syncwatch

	var com common.Command
	switch match {

	// Coin flip
	case "flip":
		com.Type = common.Flip
		com.Val = rand.Intn(2) == 0
		return com, nil

	// 8ball
	case "8ball":
		com.Type = common.EightBall

		// Select random string from the the 8ball answer array
		answers := config.GetBoardConfigs(board).Eightball
		com.Val = answers[rand.Intn(len(answers))]

		return com, nil

	// Increment pyu counter
	case "pyu":
		if !config.Get().Pyu {
			return com, nil
		}
		var err error
		com.Val, err = db.IncrementPyu()
		com.Type = common.Pyu
		return com, err

	// Return current pyu count
	case "pcount":
		if !config.Get().Pyu {
			return com, nil
		}
		var err error
		com.Val, err = db.GetPyu()
		com.Type = common.Pcount
		return com, err

	// Dice throw
	default:
		val, err := parseDice(match)
		switch err {
		case nil:
			com.Type = common.Dice
			com.Val = val
			return com, nil
		case errTooManyRolls, errDieTooBig: // Consider command invalid
			return com, nil
		default:
			return com, err
		}
	}
}

// Parse dice throw commands
func parseDice(match string) ([]uint16, error) {
	dice := diceRegexp.FindStringSubmatch(match)

	var rolls int
	if len(dice[1]) == 0 {
		rolls = 1
	} else {
		var err error
		rolls, err = strconv.Atoi(string(dice[1]))
		if err != nil {
			return nil, err
		}
		if rolls > 10 {
			return nil, errTooManyRolls
		}
	}

	max, err := strconv.Atoi(string(dice[2]))
	if err != nil {
		return nil, err
	}
	if max > 100 {
		return nil, errDieTooBig
	}

	val := make([]uint16, rolls)
	for i := 0; i < rolls; i++ {
		val[i] = uint16(rand.Intn(max)) + 1
	}

	return val, nil
}
