package parser

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
)

func TestParseLine(t *testing.T) {
	t.Parallel()

	links, com, err := ParseBody(
		[]byte("#flip,"),
		config.BoardConfigs{
			ID: "a",
		},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if links != nil {
		t.Fatalf("unexpected links: %#v", links)
	}
	if com == nil {
		t.Fatalf("no commands")
	}
	if com[0].Type != common.Flip {
		t.Fatalf("unexpected command type: %d", com[0].Type)
	}
}

func TestParseBody(t *testing.T) {
	t.Parallel()

	links, com, err := ParseBody(
		[]byte("#flip?\n>>8\n>>>6 \n(#flip)\n>foo #flip bar \n#flip"),
		config.BoardConfigs{
			ID: "a",
		},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(com); l != 3 {
		t.Errorf("unexpected command count: %d", l)
	}
	test.AssertEquals(t, links, []uint64{8, 6})
}
