package templates

import (
	"testing"

	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/types"
)

func TestBoard(t *testing.T) {
	_, err := Board("all", &types.Board{
		Threads: types.BoardThreads{
			{
				ID:      1,
				Board:   "a",
				Subject: "foo",
			},
			{
				ID:      2,
				Board:   "c",
				Subject: "bar",
				Image:   &assets.StdJPEG,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestThread(t *testing.T) {
	img := assets.StdJPEG
	img.Length = 20
	_, err := Thread(&types.Thread{
		Board:   "a",
		Subject: "foo",
		Post: types.Post{
			ID:    1,
			Image: &img,
		},
		Posts: []types.Post{
			{
				ID:   2,
				Body: "bar",
				Backlinks: types.LinkMap{
					3: {
						OP:    1,
						Board: "a",
					},
					4: {
						OP:    7,
						Board: "l",
					},
				},
			},
			{
				ID:      3,
				Body:    "foo",
				Editing: true,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
