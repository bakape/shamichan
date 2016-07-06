package parser

import (
	"reflect"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Tests) TestParseFlip(c *C) {
	comms, err := BodyParser{}.parseCommand(nil, "flip")
	c.Assert(err, IsNil)
	c.Assert(len(comms), Equals, 1)
	c.Assert(comms[0].Type, Equals, types.Flip)
	c.Assert(reflect.TypeOf(comms[0].Val).Kind(), Equals, reflect.Bool)
}

func (*Tests) TestAppendCommand(c *C) {
	var (
		first = types.Command{
			Type: types.Dice,
			Val:  []uint16{1, 2, 3},
		}
		comms = []types.Command{first}
	)

	comms, err := BodyParser{}.parseCommand(comms, "flip")
	c.Assert(err, IsNil)
	c.Assert(len(comms), Equals, 2)
	c.Assert(comms[0], DeepEquals, first)
	c.Assert(comms[1].Type, Equals, types.Flip)
	c.Assert(reflect.TypeOf(comms[1].Val).Kind(), Equals, reflect.Bool)
}

func (*Tests) TestDice(c *C) {
	samples := [...]struct {
		in         string
		isNil      bool
		rolls, max int
	}{
		{`d101`, true, 0, 0},
		{`11d100`, true, 0, 0},
		{`11d101`, true, 0, 0},
		{`d10`, false, 1, 10},
		{`10d100`, false, 10, 100},
	}
	for _, s := range samples {
		comms, err := BodyParser{}.parseCommand(nil, s.in)
		c.Assert(err, IsNil)
		if s.isNil {
			c.Assert(comms, IsNil)
		} else {
			c.Assert(len(comms), Equals, 1)
			com := comms[0]
			c.Assert(com.Type, Equals, types.Dice)
			val := com.Val.([]uint16)
			c.Assert(val[0], Equals, uint16(s.max))
			c.Assert(len(val), Equals, s.rolls+1)
		}
	}
}

func (*Tests) Test8ball(c *C) {
	answers := []string{"Yes", "No"}
	db.DBName = db.UniqueDBName()
	c.Assert(db.Connect(), IsNil)
	c.Assert(db.Exec(r.DBCreate(db.DBName)), IsNil)
	db.RSession.Use(db.DBName)
	c.Assert(db.Exec(r.TableCreate("boards")), IsNil)
	defer func() {
		c.Assert(db.Exec(r.DBDrop(db.DBName)), IsNil)
	}()
	q := r.Table("boards").Insert(config.BoardConfigs{
		ID:        "a",
		Eightball: answers,
	})
	c.Assert(db.Write(q), IsNil)

	bp := BodyParser{
		Board: "a",
	}
	comms, err := bp.parseCommand(nil, "8ball")
	c.Assert(err, IsNil)
	c.Assert(len(comms), Equals, 1)
	com := comms[0]
	c.Assert(com.Type, Equals, types.EightBall)
	val := com.Val.(string)
	if val != answers[0] && val != answers[1] {
		c.Fatalf("eightball answer not mached: %s", val)
	}
}
