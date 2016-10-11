package parser

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

func TestParseLine(t *testing.T) {
	assertTableClear(t, "boards")
	assertInsert(t, "boards", config.BoardConfigs{
		ID: "a",
	})

	t.Run("commands disabled", func(t *testing.T) {
		links, com, err := ParseLine([]byte("#flip"), "a")
		if err != nil {
			t.Fatal(err)
		}
		if links != nil {
			t.Fatalf("unexpected links: %#v", links)
		}
		AssertDeepEquals(t, com, types.Command{})
	})

	t.Run("commands enabled", func(t *testing.T) {
		q := r.Table("boards").Get("a").Update(map[string]bool{
			"hashCommands": true,
		})
		if err := db.Write(q); err != nil {
			t.Fatal(err)
		}

		links, com, err := ParseLine([]byte("#flip"), "a")
		if err != nil {
			t.Fatal(err)
		}
		if links != nil {
			t.Fatalf("unexpected links: %#v", links)
		}
		if com.Type != types.Flip {
			t.Fatalf("unexpected command type: %d", com.Type)
		}
	})
}
