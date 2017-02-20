// Hash commands such as #flip, dice and #8ball

package parser

import (
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"meguca/common"
	"meguca/config"
	"meguca/db"
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
func parseCommand(match, board string) (com common.Command, err error) {

	// TODO: #syncwatch

	switch match {

	// Coin flip
	case "flip":
		com.Type = common.Flip
		com.Val = rand.Intn(2) == 0

	// 8ball; select random string from the the 8ball answer array
	case "8ball":
		com.Type = common.EightBall
		answers := config.GetBoardConfigs(board).Eightball
		com.Val = answers[rand.Intn(len(answers))]

	// Increment pyu counter
	case "pyu":
		if config.Get().Pyu {
			com.Type = common.Pyu
			com.Val, err = db.IncrementPyu()
		}

	// Return current pyu count
	case "pcount":
		if config.Get().Pyu {
			com.Type = common.Pcount
			com.Val, err = db.GetPyu()
		}

	// Dice throw
	default:
		com.Val, err = parseDice(match)
		com.Type = common.Dice
		switch err {
		case errTooManyRolls, errDieTooBig: // Consider command invalid
			return common.Command{}, nil
		}
	}

	return
}

// Parse dice throw commands
func parseDice(match string) (val []uint16, err error) {
	dice := diceRegexp.FindStringSubmatch(match)

	var rolls int
	if len(dice[1]) == 0 {
		rolls = 1
	} else {
		rolls, err = strconv.Atoi(string(dice[1]))
		switch {
		case err != nil:
			return
		case rolls > 10:
			return nil, errTooManyRolls
		}
	}

	max, err := strconv.Atoi(string(dice[2]))
	switch {
	case err != nil:
		return
	case max > 100:
		return nil, errDieTooBig
	}

	val = make([]uint16, rolls)
	for i := 0; i < rolls; i++ {
		val[i] = uint16(rand.Intn(max)) + 1
	}
	return
}
