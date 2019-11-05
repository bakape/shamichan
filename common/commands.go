package common

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// CommandType are the various struct types of hash commands and their
// responses, such as dice rolls, #flip, #8ball, etc.
type CommandType uint8

const (
	// Dice is the dice roll command type
	Dice CommandType = iota

	// Flip is the coin flip command type
	Flip

	// EightBall is the the #8ball random answer dispenser command type
	EightBall

	// SyncWatch is the synchronized timer command type for synchronizing
	// episode time during group anime watching and such
	SyncWatch

	// Pyu - don't ask
	Pyu

	// Pcount - don't ask
	Pcount

	// Autobahn - self ban. brum brum
	Autobahn
)

// Command contains the type and value array of hash commands, such as dice
// rolls, #flip, #8ball, etc. The Val field depends on the Type field.
// Dice: []uint16
// Flip: bool
// EightBall: string
// SyncWatch: [5]uint64
// Pyu: uint64
// Pcount: uint64
type Command struct {
	Type      CommandType
	Flip      bool
	Pyu       uint64
	SyncWatch [5]uint64
	Eightball string
	Dice      []uint16
}

// MarshalJSON implements json.Marshaler
func (c Command) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, 128)
	appendStr := func(s string) {
		b = append(b, s...)
	}
	appendByte := func(c byte) {
		b = append(b, c)
	}
	appendUint := func(i uint64) {
		b = strconv.AppendUint(b, i, 10)
	}

	appendStr(`{"type":`)
	appendUint(uint64(c.Type))
	appendStr(`,"val":`)

	switch c.Type {
	case Flip:
		b = strconv.AppendBool(b, c.Flip)
	case Pyu, Pcount:
		appendUint(c.Pyu)
	case SyncWatch:
		appendByte('[')
		for i, v := range c.SyncWatch {
			if i != 0 {
				appendByte(',')
			}
			appendUint(v)
		}
		appendByte(']')
	case EightBall:
		b = strconv.AppendQuote(b, c.Eightball)
	case Dice:
		appendByte('[')
		for i, v := range c.Dice {
			if i != 0 {
				appendByte(',')
			}
			appendUint(uint64(v))
		}
		appendByte(']')
	}

	b = append(b, '}')

	return b, nil
}

// UnmarshalJSON decodes a dynamically-typed JSON-encoded command into the
// statically-typed Command struct
func (c *Command) UnmarshalJSON(data []byte) error {
	if len(data) < 18 {
		return fmt.Errorf("data too short: %s", string(data))
	}

	typ, err := strconv.ParseUint(string(data[8]), 10, 8)
	if err != nil {
		return err
	}

	data = data[16 : len(data)-1]
	switch CommandType(typ) {
	case Flip:
		c.Type = Flip
		err = json.Unmarshal(data, &c.Flip)
	case Pyu:
		c.Type = Pyu
		err = json.Unmarshal(data, &c.Pyu)
	case Pcount:
		c.Type = Pcount
		err = json.Unmarshal(data, &c.Pyu)
	case SyncWatch:
		c.Type = SyncWatch
		err = json.Unmarshal(data, &c.SyncWatch)
	case EightBall:
		c.Type = EightBall
		err = json.Unmarshal(data, &c.Eightball)
	case Dice:
		c.Type = Dice
		err = json.Unmarshal(data, &c.Dice)
	case Autobahn:
		c.Type = Autobahn
	default:
		return fmt.Errorf("unknown command type: %d", typ)
	}
	return err
}
