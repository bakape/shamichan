// Hash commands such as #flip, dice and #8ball

package parser

import (
	"bytes"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
)

var (
	diceRegexp = regexp.MustCompile(`(\d*)d(\d+)`)

	errTooManyRolls = diceError(0)
	errDieTooBig    = diceError(1)

	flipCommand      = []byte("flip")
	eightballCommand = []byte("8ball")
)

type diceError int

func (d diceError) Error() string {
	return "dice error: " + strconv.Itoa(int(d))
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Parse a matched hash command
func parseCommand(match []byte, board string) (types.Command, error) {

	// TODO: #pyu, #queue and #syncwatch

	var com types.Command
	switch {
	case bytes.Equal(match, flipCommand):
		com.Type = types.Flip
		com.Val = rand.Intn(2) > 1
		return com, nil
	case bytes.Equal(match, eightballCommand):
		com.Type = types.EightBall

		// Select random string from the the 8ball answer array
		q := db.
			GetBoardConfig(board).
			Field("eightball").
			Sample(1).
			AtIndex(0)
		err := db.One(q, &com.Val)
		return com, err
	default:
		val, err := parseDice(match)
		switch err {
		case nil:
			com.Type = types.Dice
			com.Val = val
			return com, nil
		case errTooManyRolls, errDieTooBig: // Consider command invalid
			return com, nil
		default:
			return com, err
		}
	}
}

// Parse dice thow commands
func parseDice(match []byte) ([]uint16, error) {
	dice := diceRegexp.FindSubmatch(match)

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
		val[i] = uint16(rand.Intn(max))
	}

	return val, nil
}
