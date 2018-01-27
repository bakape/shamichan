package common

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mailru/easyjson/jwriter"
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

	// Russian Roulette, first poster has a 1/6 chance of dying, then 1/5, etc
	// resets to 1/6 chance after someone dies
	Roulette

	// Rcount - number of bans handed out from #roulette
	Rcount
)

// Command contains the type and value array of hash commands, such as dice
// rolls, #flip, #8ball, etc. The Val field depends on the Type field.
// Dice: []uint16
// Flip: bool
// EightBall: string
// SyncWatch: [5]uint64
// Pyu: uint64
// Pcount: uint64
// Roulette: [2]uint8
// Rcount: uint64
type Command struct {
	Type      CommandType
	Flip      bool
	Pyu       uint64
	SyncWatch [5]uint64
	Eightball string
	Dice      []uint16
	Roulette  [2]uint8
}

// MarshalJSON implements json.Marshaler
func (c Command) MarshalJSON() ([]byte, error) {
	var w jwriter.Writer
	c.MarshalEasyJSON(&w)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON implements easyjson.Marshaler. Defined manually to
// dynamically marshal the appropriate fields by struct type.
func (c Command) MarshalEasyJSON(w *jwriter.Writer) {
	w.RawString(`{"type":`)
	w.Uint8(uint8(c.Type))
	w.RawString(`,"val":`)

	switch c.Type {
	case Flip:
		w.Bool(c.Flip)
	case Pyu, Pcount, Rcount:
		w.Uint64(c.Pyu)
	case SyncWatch:
		w.RawByte('[')
		for i, v := range c.SyncWatch {
			if i != 0 {
				w.RawByte(',')
			}
			w.Uint64(v)
		}
		w.RawByte(']')
	case EightBall:
		w.String(c.Eightball)
	case Dice:
		w.RawByte('[')
		for i, v := range c.Dice {
			if i != 0 {
				w.RawByte(',')
			}
			w.Uint16(v)
		}
		w.RawByte(']')
	case Roulette:
		w.RawByte('[')
		for i, v := range c.Roulette {
			if i != 0 {
				w.RawByte(',')
			}
			w.Uint8(v)
		}
		w.RawByte(']')
	}

	w.RawByte('}')
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
	case Roulette:
		c.Type = Roulette
		err = json.Unmarshal(data, &c.Roulette)
	case Rcount:
		c.Type = Rcount
		err = json.Unmarshal(data, &c.Pyu)
	default:
		return fmt.Errorf("unknown command type: %d", typ)
	}
	return err
}
