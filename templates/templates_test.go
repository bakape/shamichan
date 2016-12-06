package templates

import (
	"path/filepath"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/util"
)

func init() {
	_, err := config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})
	if err != nil {
		panic(err)
	}
	config.Set(config.Configs{})
	lang.Dir = filepath.Join("..", "lang")

	fns := []func() error{lang.Load, Compile}
	if err := util.Waterfall(fns); err != nil {
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
	_, err := Board("all", lang.Packs["en_GB"], true, common.Board{
		Threads: common.BoardThreads{
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
