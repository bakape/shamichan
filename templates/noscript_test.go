package templates

import "testing"
import "github.com/bakape/meguca/types"
import "github.com/bakape/meguca/imager/assets"

func TestBoard(t *testing.T) {
	html, err := Board("all", &types.Board{
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
	t.Log(string(html))
}
