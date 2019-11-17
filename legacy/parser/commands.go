// Hash commands such as #flip, dice and #8ball

package parser

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bakape/meguca/common"
)

var (
	syncWatchRegexp = regexp.MustCompile(`^sw(\d+:)?(\d+):(\d+)([+-]\d+)?$`)

	errTooManyRolls = common.ErrInvalidInput("too many rolls")
	errDieTooBig    = common.ErrInvalidInput("die too big")

	// TODO: Per-thread 8ball config
	eightBallAnswers = [...]string{
		"Yes",
		"No",
		"Maybe",
		"It can't be helped",
		"Hell yeah, motherfucker!",
		"Anta baka?",
	}
)

// Returns a cryptographically secure pseudorandom int in the interval [0;max)
func randInt(max int) int {
	i, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if i == nil { // Fuck error reporting here
		return 0
	}
	return int(i.Int64())
}

// Parse a matched hash command
func parseCommand(match []byte) (com common.Command, err error) {
	switch {

	// Coin flip
	case bytes.Equal(match, []byte("flip")):
		com.Type = common.Flip
		com.Flip = randInt(2) == 1

	// 8ball; select random string from the the 8ball answer array
	case bytes.Equal(match, []byte("8ball")):
		com.Type = common.EightBall
		com.Eightball = eightBallAnswers[randInt(len(eightBallAnswers))]

	// Increment pyu counter
	case bytes.Equal(match, []byte("pyu")):
		com.Type = common.Pyu
		// DB queries deferred to post close

	// Return current pyu count
	case bytes.Equal(match, []byte("pcount")):
		com.Type = common.Pcount
		// DB queries deferred to post close

	default:
		matchStr := string(match)

		// Synchronized time counter
		if strings.HasPrefix(matchStr, "sw") {
			com.Type = common.SyncWatch
			com.SyncWatch = parseSyncWatch(matchStr)
			return
		}

		// Dice throw
		com.Type = common.Dice
		com.Dice, err = parseDice(matchStr)
	}

	return
}

func isNumError(err error) bool {
	_, ok := err.(*strconv.NumError)
	return ok
}

// Parse dice throw commands
func parseDice(match string) (val []uint16, err error) {
	dice := common.DiceRegexp.FindStringSubmatch(match)

	var rolls int
	if len(dice[1]) == 0 {
		rolls = 1
	} else {
		rolls, err = strconv.Atoi(string(dice[1]))
		switch {
		case err != nil:
			if isNumError(err) {
				err = common.StatusError{
					Err:  err,
					Code: 400,
				}
			}
			return
		case rolls > 10:
			return nil, errTooManyRolls
		}
	}

	max, err := strconv.Atoi(string(dice[2]))
	switch {
	case err != nil:
		if isNumError(err) {
			err = common.StatusError{
				Err:  err,
				Code: 400,
			}
		}
		return
	case max > common.MaxDiceSides:
		return nil, errDieTooBig
	}

	val = make([]uint16, rolls)
	for i := 0; i < rolls; i++ {
		if max != 0 {
			val[i] = uint16(randInt(max)) + 1
		} else {
			val[i] = 0
		}
	}
	return
}

func parseSyncWatch(match string) [5]uint64 {
	m := syncWatchRegexp.FindStringSubmatch(match)
	var (
		hours, min, sec, offset uint64
		offsetDirection         byte
	)

	if m[1] != "" {
		hours, _ = strconv.ParseUint(m[1][:len(m[1])-1], 10, 64)
	}
	min, _ = strconv.ParseUint(m[2], 10, 64)
	sec, _ = strconv.ParseUint(m[3], 10, 64)
	if m[4] != "" {
		offsetDirection = m[4][0]
		offset, _ = strconv.ParseUint(m[4][1:], 10, 64)
	}

	start := uint64(time.Now().Unix())
	switch offsetDirection {
	case '+':
		start += offset
	case '-':
		start -= offset
	}
	end := start + sec + (hours*60+min)*60

	return [5]uint64{
		hours,
		min,
		sec,
		start,
		end,
	}
}
