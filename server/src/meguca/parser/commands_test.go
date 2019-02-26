package parser

import (
	"database/sql"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	. "meguca/test"
	"meguca/test/test_db"
	"testing"
)

func TestFlip(t *testing.T) {
	var isSlut bool
	t.Parallel()

	com, err := parseCommand([]byte("flip"), "a", 1, 1, "::1", &isSlut)
	if err != nil {
		t.Fatal(err)
	}
	if com.Type != common.Flip {
		t.Fatalf("unexpected command type: %d", com.Type)
	}
}

func TestDice(t *testing.T) {
	var isSlut bool
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
			com, err := parseCommand([]byte(c.in), "a", 1, 1, "::1", &isSlut)
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
	var isSlut bool
	answers := []string{"Yes", "No"}
	config.SetBoardConfigs(config.BoardConfigs{
		ID:        "a",
		Eightball: answers,
	})

	com, err := parseCommand([]byte("8ball"), "a", 1, 1, "::1", &isSlut)
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
	var isSlut bool
	test_db.ClearTables(t, "boards", "pyu", "pyu_limit")
	writeSampleBoard(t)
	writeSampleThread(t)

	err := db.SetPcount(0)
	if err != nil {
		t.Fatal(err)
	}
	err = db.WritePyu("a")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("disabled", func(t *testing.T) {
		config.SetBoardConfigs(config.BoardConfigs{
			BoardPublic: config.BoardPublic{
				Pyu: false,
			},
			ID: "a",
		})

		for _, in := range [...]string{"pyu", "pcount"} {
			com, err := parseCommand([]byte(in), "a", 1, 1, "::1", &isSlut)
			if err != nil {
				t.Error(err)
			}
			AssertDeepEquals(t, com, common.Command{
				Type: com.Type,
			})
		}
	})

	t.Run("enabled", func(t *testing.T) {
		config.SetBoardConfigs(config.BoardConfigs{
			BoardPublic: config.BoardPublic{
				Pyu: true,
			},
			ID: "a",
		})

		cases := [...]struct {
			name, in string
			Type     common.CommandType
			Val      uint64
		}{
			{"count on zero", "pcount", common.Pcount, 0},
			{"increment", "pyu", common.Pyu, 1},
			{"count", "pcount", common.Pcount, 1},
			{"increment with limit set", "pyu", common.Pyu, 2},
			{"increment with limit set", "pyu", common.Pyu, 3},
			{"increment with limit set", "pyu", common.Pyu, 4},
			{"pyu limit reached", "pyu", common.Pyu, 4},
		}

		for i := range cases {
			c := cases[i]
			t.Run(c.name, func(t *testing.T) {
				com, err := parseCommand([]byte(c.in), "a", 1, 1, "::1", &isSlut)
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

	t.Run("expire limit", func(t *testing.T) {
		err := db.InTransaction(false, func(tx *sql.Tx) error {
			return db.FreePyuLimit()
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
