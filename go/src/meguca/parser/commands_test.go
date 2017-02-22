package parser

import (
	"meguca/common"
	"meguca/config"
	"meguca/db"
	. "meguca/test"
	"testing"
)

func TestFlip(t *testing.T) {
	t.Parallel()

	com, err := parseCommand([]byte("flip"), "a")
	if err != nil {
		t.Fatal(err)
	}
	if com.Type != common.Flip {
		t.Fatalf("unexpected command type: %d", com.Type)
	}
}

func TestDice(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in   string
		err        error
		rolls, max int
	}{
		{"too many sides", `d101`, errDieTooBig, 0, 0},
		{"too many dice", `11d100`, errTooManyRolls, 0, 0},
		{"valid single die", `d10`, nil, 1, 10},
		{"valid multiple dice", `10d100`, nil, 10, 100},
	}
	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			com, err := parseCommand([]byte(c.in), "a")
			if err != c.err {
				t.Fatalf("unexpected error: %s : %s", c.err, err)
			} else {
				if com.Type != common.Dice {
					t.Fatalf("unexpected command type: %d", com.Type)
				}
				if l := len(com.Dice); l != c.rolls {
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
	if com.Type != common.EightBall {
		t.Fatalf("unexpected command type: %d", com.Type)
	}
	val := com.Eightball
	if val != answers[0] && val != answers[1] {
		t.Fatalf("unexpected answer: %s", val)
	}
}

func TestPyu(t *testing.T) {
	if err := db.SetPyu(0); err != nil {
		t.Fatal(err)
	}

	t.Run("disabled", func(t *testing.T) {
		(*config.Get()).Pyu = false
		for _, in := range [...]string{"pyu", "pcount"} {
			com, err := parseCommand([]byte(in), "a")
			if err != nil {
				t.Error(err)
			}
			AssertDeepEquals(t, com, common.Command{})
		}
	})

	t.Run("enabled", func(t *testing.T) {
		(*config.Get()).Pyu = true

		cases := [...]struct {
			name, in string
			Type     common.CommandType
			Val      uint64
		}{
			{"count on zero", "pcount", common.Pcount, 0},
			{"increment", "pyu", common.Pyu, 1},
			{"count", "pcount", common.Pcount, 1},
		}

		for i := range cases {
			c := cases[i]
			t.Run(c.name, func(t *testing.T) {
				com, err := parseCommand([]byte(c.in), "a")
				if err != nil {
					t.Fatal(err)
				}
				AssertDeepEquals(t, com, common.Command{
					Type: c.Type,
					Pyu:  c.Val,
				})
			})
		}
	})
}
