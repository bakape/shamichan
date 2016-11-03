package templates

import "testing"
import "github.com/bakape/meguca/types"
import "github.com/bakape/meguca/imager/assets"

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
	html, err := Thread(&types.Thread{
		Board:   "a",
		Subject: "foo",
		Post: types.Post{
			ID:    1,
			Image: &assets.StdJPEG,
		},
		Posts: []types.Post{
			{
				ID:   2,
				Body: "foo",
			},
			{
				ID:   3,
				Body: "bar",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(html))
}
