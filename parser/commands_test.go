package parser

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
)

func TestFlip(t *testing.T) {
	t.Parallel()

	com, err := parseCommand([]byte("flip"), config.BoardConfigs{})
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
		{"too many sides", `d10001`, errDieTooBig, 0, 0},
		{"too many dice", `11d100`, errTooManyRolls, 0, 0},
		{"valid single die", `d10`, nil, 1, 10},
		{"valid multiple dice", `10d100`, nil, 10, 100},
	}
	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			com, err := parseCommand([]byte(c.in), config.BoardConfigs{})
			if err != c.err {
				t.Fatalf("unexpected error: %s : %s", c.err, err)
			} else {
				if com.Type != common.Dice {
					t.Fatalf("unexpected command type: %d", com.Type)
				}
				if l := len(com.Dice); l != c.rolls {
					test.LogUnexpected(t, c.rolls, l)
				}
			}
		})
	}
}

func Test8ball(t *testing.T) {
	t.Parallel()

	answers := []string{"Yes", "No"}

	com, err := parseCommand(
		[]byte("8ball"),
		config.BoardConfigs{
			ID:        "a",
			Eightball: answers,
		},
	)

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
