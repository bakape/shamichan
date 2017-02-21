// Hash commands such as #flip, dice and #8ball

package parser

import (
	"errors"
	"math/rand"
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

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Parse a matched hash command
func parseCommand(match, board string) (com common.Command, err error) {
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

	default:
		// Synchronized time counter
		if strings.HasPrefix(match, "sw") {
			com.Type = common.SyncWatch
			com.Val = parseSyncWatch(match)
			return
		}

		// Dice throw
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
	case max > 100:
		return nil, errDieTooBig
	}

	val = make([]uint16, rolls)
	for i := 0; i < rolls; i++ {
		val[i] = uint16(rand.Intn(max)) + 1
	}
	return
}

func parseSyncWatch(match string) interface{} {
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
