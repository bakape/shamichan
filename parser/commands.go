// Hash commands such as #flip, dice and #8ball

package parser

import (
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/bakape/meguca/types"

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
func (b BodyParser) parseCommand(commands []types.Command, match string) (
	[]types.Command, error,
) {

	// TODO: #pyu, #queue and #syncwatch

	var com types.Command
	switch match {
	case "flip":
		com.Type = types.Flip
		com.Val = rand.Intn(2) > 1
	case "8ball":
		com.Type = types.EightBall

		// Select random string from the the 8ball answer array
		q := db.
			GetBoardConfig(b.Board).
			Field("eightball").
			Sample(1).
			AtIndex(0)
		if err := db.One(q, &com.Val); err != nil {
			return nil, err
		}
	default:
		val, err := parseDice(match)
		switch err {
		case nil:
			com.Type = types.Dice
			com.Val = val
		// Consider command invalid
		case errTooManyRolls, errDieTooBig:
			return commands, nil
		default:
			return nil, err
		}
	}

	if commands == nil {
		return []types.Command{com}, nil
	}
	return append(commands, com), nil
}

// Parse dice thow commands
func parseDice(match string) ([]uint16, error) {
	dice := diceRegexp.FindStringSubmatch(match)

	var rolls int
	if dice[1] == "" {
		rolls = 1
	} else {
		var err error
		rolls, err = strconv.Atoi(dice[1])
		if err != nil {
			return nil, err
		}
		if rolls > 10 {
			return nil, errTooManyRolls
		}
	}

	max, err := strconv.Atoi(dice[2])
	if err != nil {
		return nil, err
	}
	if max > 100 {
		return nil, errDieTooBig
	}

	val := make([]uint16, rolls+1)
	val[0] = uint16(max)
	for i := 1; i < rolls+1; i++ {
		val[i] = uint16(rand.Intn(max))
	}

	return val, nil
}
