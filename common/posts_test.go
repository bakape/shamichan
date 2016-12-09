package common

import (
	"encoding/json"
	"testing"

	. "github.com/bakape/meguca/test"
)

func TestMarshalUnmarshalCommands(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name string
		typ  CommandType
		val  interface{}
		res  string
	}{
		{"dice", Dice, []uint16{100, 50, 50}, `{"type":0,"val":[100,50,50]}`},
		{"flip", Flip, true, `{"type":1,"val":true}`},
		{"8ball", EightBall, "Yes", `{"type":2,"val":"Yes"}`},
		{"pyu", Pyu, 999, `{"type":4,"val":999}`},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			command := Command{
				Type: c.typ,
				Val:  c.val,
			}
			data, err := json.Marshal(command)
			if err != nil {
				t.Fatal(err)
			}
			if s := string(data); s != c.res {
				LogUnexpected(t, c.res, s)
			}

			var val Command
			if err := json.Unmarshal(data, &val); err != nil {
				t.Fatal(err)
			}
			if val.Type != c.typ {
				LogUnexpected(t, c.typ, val.Type)
			}
		})
	}
}
