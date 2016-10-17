package parser

import (
	"reflect"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
)

func TestFlip(t *testing.T) {
	t.Parallel()

	com, err := parseCommand([]byte("flip"), "a")
	if err != nil {
		t.Fatal(err)
	}
	if com.Type != types.Flip {
		t.Fatalf("unexpected command type: %d", com.Type)
	}
	if k := reflect.TypeOf(com.Val).Kind(); k != reflect.Bool {
		t.Fatalf("unexpected value kind: %d", k)
	}
}

func TestDice(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in   string
		isNil      bool
		rolls, max int
	}{
		{"too many sides", `d101`, true, 0, 0},
		{"too many dice", `11d100`, true, 0, 0},
		{"too many dice and sides", `11d101`, true, 0, 0},
		{"valid single die", `d10`, false, 1, 10},
		{"valid multiple dice", `10d100`, false, 10, 100},
	}
	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			com, err := parseCommand([]byte(c.in), "a")
			if err != nil {
				t.Fatal(err)
			}
			if c.isNil {
				if com.Val != nil {
					t.Fatalf("unexpected value: %#v", com.Val)
				}
			} else {
				if com.Type != types.Dice {
					t.Fatalf("unexpected command type: %d", com.Type)
				}
				val := com.Val.([]uint16)
				if l := len(val); l != c.rolls {
					LogUnexpected(t, c.rolls, l)
				}
			}
		})
	}
}

func Test8ball(t *testing.T) {
	answers := []string{"Yes", "No"}
	config.SetBoardConfigs(config.BoardConfigs{
		ID:        "a",
		Eightball: answers,
	})

	com, err := parseCommand([]byte("8ball"), "a")
	if err != nil {
		t.Fatal(err)
	}
	if com.Type != types.EightBall {
		t.Fatalf("unexpected command type: %d", com.Type)
	}
	val := com.Val.(string)
	if val != answers[0] && val != answers[1] {
		t.Fatalf("unexpected answer: %s", val)
	}
}

func TestPyu(t *testing.T) {
	assertTableClear(t, "main")
	assertInsert(t, "main", db.Document{ID: "info"})

	t.Run("disabled", func(t *testing.T) {
		(*config.Get()).Pyu = false
		for _, in := range [...][]byte{pyuCommand, pcountCommand} {
			com, err := parseCommand(in, "a")
			if err != nil {
				t.Error(err)
			}
			AssertDeepEquals(t, com, types.Command{})
		}
	})

	t.Run("enabled", func(t *testing.T) {
		(*config.Get()).Pyu = true

		cases := [...]struct {
			name string
			in   []byte
			Type types.CommandType
			Val  int
		}{
			{"count on zero", pcountCommand, types.Pcount, 0},
			{"increment", pyuCommand, types.Pyu, 1},
			{"count", pcountCommand, types.Pcount, 1},
		}

		for i := range cases {
			c := cases[i]
			t.Run(c.name, func(t *testing.T) {
				com, err := parseCommand(c.in, "a")
				if err != nil {
					t.Fatal(err)
				}
				AssertDeepEquals(t, com, types.Command{
					Type: c.Type,
					Val:  c.Val,
				})
			})
		}
	})
}
