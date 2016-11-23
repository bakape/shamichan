package templates

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/lang"
)

func TestThread(t *testing.T) {
	img := assets.StdJPEG
	img.Length = 20
	_, err := Thread(lang.Packs["en_GB"], true, common.Thread{
		Board:   "a",
		Subject: "foo",
		Post: common.Post{
			ID:    1,
			Image: &img,
		},
		Posts: []common.Post{
			{
				ID:   2,
				Body: "bar",
				Backlinks: common.LinkMap{
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
