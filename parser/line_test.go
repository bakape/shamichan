package parser

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
)

func TestParseLine(t *testing.T) {
	config.SetBoardConfigs(config.BoardConfigs{
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
		config.SetBoardConfigs(config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				PostParseConfigs: config.PostParseConfigs{
					HashCommands: true,
				},
			},
		})

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
