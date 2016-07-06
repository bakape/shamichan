package types

import (
	"encoding/json"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Tests struct{}

var _ = Suite(&Tests{})

func (*Tests) TestMarshalUnmarshalCommands(c *C) {
	samples := [...]struct {
		typ CommandType
		val interface{}
		res string
	}{
		{Dice, []uint16{100, 50, 50}, `{"type":0,"val":[100,50,50]}`},
		{Flip, true, `{"type":1,"val":true}`},
		{EightBall, "Yes", `{"type":2,"val":"Yes"}`},
		{Pyu, 999, `{"type":4,"val":999}`},
	}
	for _, s := range samples {
		command := Command{
			Type: s.typ,
			Val:  s.val,
		}
		data, err := json.Marshal(command)
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, s.res)

		var val Command
		c.Assert(json.Unmarshal(data, &val), IsNil)
		c.Assert(val.Type, Equals, s.typ)
	}
}
