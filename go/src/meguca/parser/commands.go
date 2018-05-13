// Hash commands such as #flip, dice and #8ball

package parser

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math/big"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	syncWatchRegexp = regexp.MustCompile(`^sw(\d+:)?(\d+):(\d+)([+-]\d+)?$`)

	errTooManyRolls = errors.New("too many rolls")
	errDieTooBig    = errors.New("die too big")
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
func parseCommand(match []byte, board string, id uint64, ip string) (com common.Command, err error) {
	boardConfig := config.GetBoardConfigs(board)
	
	switch {

	// Coin flip
	case bytes.Equal(match, []byte("flip")):
		com.Type = common.Flip
		com.Flip = randInt(2) == 1

	// 8ball; select random string from the the 8ball answer array
	case bytes.Equal(match, []byte("8ball")):
		com.Type = common.EightBall
		answers := boardConfig.Eightball
		if len(answers) != 0 {
			com.Eightball = answers[randInt(len(answers))]
		}

	// Increment pyu counter
	case bytes.Equal(match, []byte("pyu")):
		com.Type = common.Pyu

		if boardConfig.Pyu {
			now := time.Now().UTC()
			tx, err := db.StartTransaction()
			
			if err != nil {
				return com, err
			}

			defer db.RollbackOnError(tx, &err)
			exists, err := db.PyuLimitExists(tx, ip, board)

			if err != nil {
				return com, err
			}

			if !exists {
				err = db.WritePyuLimit(tx, ip, board)

				if err != nil {
					return com, err
				}
			}

			limit, err := db.GetPyuLimit(tx, ip, board)

			if err != nil {
				return com, err
			}
			
			expires, err := db.GetPyuLimitExpires(tx, ip, board)

			if err != nil {
				return com, err
			}

			if limit == 1 {
				err = db.SetPyuLimitExpires(tx, ip, board)

				if err != nil {
					return com, err
				}
			} else if limit == 0 && expires.Before(now) {
				limit, err = db.ResetPyuLimit(tx, ip, board)

				if err != nil {
					return com, err
				}
			}

			if limit == 0 {
				com.Pyu, err = db.GetPcountA(tx, board)

				if err != nil {
					return com, err
				}

				err = db.Ban(board, "stop being such a slut", "system",
								now.Add(time.Second*30), false, id)

				if err != nil {
					return com, err
				}
			} else {
				com.Pyu, err = db.IncrementPcount(tx, board)

				if err != nil {
					return com, err
				}

				err = db.DecrementPyuLimit(tx, ip, board)

				if err != nil {
					return com, err
				}
			}

			err = tx.Commit()
		} else {
			com.Pyu, err = db.GetPcount(board)
		}

	// Return current pyu count
	case bytes.Equal(match, []byte("pcount")):
		com.Type = common.Pcount
		com.Pyu, err = db.GetPcount(board)

	// Roulette
	case bytes.Equal(match, []byte("roulette")):
		com.Type = common.Roulette
		var max uint8
		max, err = db.DecrementRoulette()
		if err != nil {
			return
		}
		roll := uint8(randInt(int(max)) + 1)
		if roll == 1 {
			err = db.ResetRoulette()
		}
		com.Roulette = [2]uint8{roll, max}

	// Return current roulette count
	case bytes.Equal(match, []byte("rcount")):
		com.Type = common.Rcount
		com.Pyu, err = db.GetRcount()

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
			return
		case rolls > 10:
			return nil, errTooManyRolls
		}
	}

	max, err := strconv.Atoi(string(dice[2]))
	switch {
	case err != nil:
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
