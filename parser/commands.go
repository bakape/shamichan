// Hash commands such as #flip, dice and #8ball

package parser

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
)

var (
	syncWatchRegexp = regexp.MustCompile(`^sw(\d+:)?(\d+):(\d+)([+-]\d+)?$`)

	errTooManyRolls = common.ErrInvalidInput("too many rolls")
	errDieTooBig    = common.ErrInvalidInput("die too big")
)

const (
	dubs = iota + 1
	trips
	quads
)

var rollModifiers = [...]struct {
	re   *regexp.Regexp
	list []uint16
}{
	{
		re: regexp.MustCompile(`(?i)\bdubs\b`),
	},
	{
		re: regexp.MustCompile(`(?i)\btrips\b`),
	},
	{
		re: regexp.MustCompile(`(?i)\bquads\b`),
	},
}

func init() {
	for i := range rollModifiers {
		var base uint16
		switch i {
		case 0:
			base = 11
		case 1:
			base = 111
		case 2:
			base = 1111
		}
		rollModifiers[i].list = append(rollModifiers[i].list, 1)
		for m := uint16(1); m <= uint16(9); m++ {
			rollModifiers[i].list = append(rollModifiers[i].list, base*m)
		}
	}
}

// Returns a cryptographically secure pseudorandom int in the interval [0;max)
func randInt(max int) int {
	i, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if i == nil { // Fuck error reporting here
		return 0
	}
	return int(i.Int64())
}

// Parse a matched hash command
func parseCommand(
	match []byte,
	board string,
	thread uint64,
	id uint64,
	ip string,
	isSlut *bool,
	isDead *bool,
	body []byte,
) (
	com common.Command, err error,
) {
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
			err = db.InTransaction(false, func(tx *sql.Tx) (err error) {
				exists, err := db.PyuLimitExists(tx, ip, board)

				if err != nil {
					return
				}

				if !exists {
					err = db.WritePyuLimit(tx, ip, board)

					if err != nil {
						return
					}
				}

				limit, err := db.GetPyuLimit(tx, ip, board)

				if err != nil {
					return
				}

				restricted, err := db.GetPyuLimitRestricted(tx, ip, board)

				if err != nil {
					return
				}

				if restricted {
					com.Pyu, err = db.GetPcountA(tx, board)

					if err != nil {
						return
					}

					if !*isSlut {
						*isSlut = true
						err = db.Ban(board, "stop being such a slut", "system",
							time.Hour, id)
					}

					if err != nil {
						return
					}
				} else {
					switch limit {
					case 1:
						err = db.SetPyuLimitRestricted(tx, ip, board)

						if err != nil {
							return
						}

						fallthrough
					default:
						com.Pyu, err = db.IncrementPcount(tx, board)

						if err != nil {
							return
						}

						err = db.DecrementPyuLimit(tx, ip, board)

						if err != nil {
							return
						}
					}
				}

				return
			})
		} else {
			com.Pyu, err = db.GetPcount(board)
		}

	// Return current pyu count
	case bytes.Equal(match, []byte("pcount")):
		com.Type = common.Pcount
		com.Pyu, err = db.GetPcount(board)

	// Autobahn
	case bytes.Equal(match, []byte("autobahn")):
		com.Type = common.Autobahn
		if !*isDead {
			*isDead = true
			err = db.Ban(board, "brum brum", "system", time.Hour, id)
		}

	default:
		matchStr := string(match)

		// Synchronized time counter
		if strings.HasPrefix(matchStr, "sw") {
			com.Type = common.SyncWatch
			com.SyncWatch = parseSyncWatch(matchStr)
			return
		}

		var mod int
		for i := len(rollModifiers) - 1; i >= 0; i-- {
			m := rollModifiers[i]
			if m.re.Match(body) {
				mod = i + 1
				break
			}
		}

		// Dice throw
		com.Type = common.Dice
		com.Dice, err = parseDice(matchStr, mod)
	}

	return
}

func isNumError(err error) bool {
	_, ok := err.(*strconv.NumError)
	return ok
}

// Parse dice throw commands
func parseDice(match string, mod int) (val []uint16, err error) {
	dice := common.DiceRegexp.FindStringSubmatch(match)

	var rolls int
	if len(dice[1]) == 0 {
		rolls = 1
	} else {
		rolls, err = strconv.Atoi(string(dice[1]))
		switch {
		case err != nil:
			if isNumError(err) {
				err = common.StatusError{err, 400}
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
			err = common.StatusError{err, 400}
		}
		return
	case max > common.MaxDiceSides:
		return nil, errDieTooBig
	}

	var overrides []uint16
	if mod != 0 && randInt(3) == 0 {
		i := 0
		list := rollModifiers[mod-1].list
		for roll := list[i]; i < len(list) && roll < uint16(max); i++ {
		}
		overrides = list[:i+1]
	}

	val = make([]uint16, rolls)
	for i := 0; i < rolls; i++ {
		if max != 0 {
			if len(overrides) > 1 {
				val[i] = overrides[randInt(len(overrides))]
			} else {
				val[i] = uint16(randInt(max)) + 1
			}
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
