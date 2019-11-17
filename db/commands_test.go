package db

import (
	"bytes"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/test"
)

func TestPopulateCommands(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP: 1,
			Post: common.Post{
				Editing: true,
			},
		},
		IP: "::1",
	}
	insertPost(t, &p)

	err := ClosePost(p.ID, "a", "", nil, []common.Command{
		{
			Type: common.Flip,
			Flip: true,
		},
		{
			Type: common.Pyu,
		},
		{
			Type: common.Pcount,
		},
		{
			Type: common.Pyu,
		},
		{
			Type: common.Pcount,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	buf, err := GetPost(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertJSON(t, bytes.NewReader(buf), common.StandalonePost{
		OP: 1,
		Post: common.Post{
			ID:   p.ID,
			Time: p.Time,
			Commands: []common.Command{
				{
					Type: common.Flip,
					Flip: true,
				},
				{
					Type: common.Pyu,
					Pyu:  1,
				},
				{
					Type: common.Pcount,
					Pyu:  1,
				},
				{
					Type: common.Pyu,
					Pyu:  2,
				},
				{
					Type: common.Pcount,
					Pyu:  2,
				},
			},
		},
	})
}
