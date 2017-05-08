package templates

import (
	"testing"

	"meguca/common"
	"meguca/config"
	"meguca/imager/assets"
	"meguca/lang"
	"meguca/util"
)

func init() {
	_, err := config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})
	if err != nil {
		panic(err)
	}
	config.Set(config.Configs{})

	if err := util.Waterfall(lang.Load, Compile); err != nil {
		panic(err)
	}
}

func TestCompileTemplates(t *testing.T) {
	config.SetClient([]byte{1}, "hash")
	(*config.Get()).Captcha = true

	if err := Compile(); err != nil {
		t.Fatal(err)
	}
}

func TestBoard(t *testing.T) {
	board := common.Board{
		{
			Post: common.Post{
				ID: 1,
			},
			Board:   "a",
			Subject: "foo",
		},
		{
			Post: common.Post{
				ID:    2,
				Image: &assets.StdJPEG,
			},
			Board:   "c",
			Subject: "bar",
		},
	}
	html := CatalogThreads(board, nil)

	Board("all", lang.Packs["en_GB"], false, false, []byte(html))
}

func TestThread(t *testing.T) {
	img := assets.StdJPEG
	img.Length = 20
	thread := common.Thread{
		Board:   "a",
		Subject: "foo",
		Post: common.Post{
			ID:    1,
			Image: &img,
		},
		Abbrev: true,
		Posts: []common.Post{
			{
				ID:   2,
				Body: "bar",
				Backlinks: [][2]uint64{
					{3, 1},
					{4, 7},
				},
			},
			{
				ID:      3,
				Body:    "foo",
				Editing: true,
			},
		},
	}

	html := ThreadPosts(thread, nil)
	Thread(lang.Packs["en_GB"], 1, "a", true, []byte(html))
}
