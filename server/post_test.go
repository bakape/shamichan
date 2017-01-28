package server

import (
	"strconv"
	"testing"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
)

func TestSpoilerImage(t *testing.T) {
	assertTableClear(t, "boards", "images")
	writeSampleBoard(t)
	writeSampleThread(t)
	writeSampleImage(t)

	const password = "123"
	hash, err := auth.BcryptHash(password, 6)
	if err != nil {
		t.Fatal(err)
	}

	posts := [...]db.DatabasePost{
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 2,
					Image: &common.Image{
						ImageCommon: assets.StdJPEG.ImageCommon,
					},
				},
				OP: 1,
			},
		},
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 3,
				},
				OP: 1,
			},
		},
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 4,
					Image: &common.Image{
						ImageCommon: assets.StdJPEG.ImageCommon,
						Spoiler:     true,
					},
				},
				OP: 1,
			},
		},
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 5,
					Image: &common.Image{
						ImageCommon: assets.StdJPEG.ImageCommon,
					},
				},
				OP: 1,
			},
		},
	}
	for _, p := range posts {
		if err := db.WritePost(nil, p); err != nil {
			t.Fatal(err)
		}
	}

	cases := [...]struct {
		name                string
		id                  uint64
		password            string
		code                int
		hasImage, spoilered bool
	}{
		{"no image", 3, password, 400, false, false},
		{"wrong password", 5, "122", 403, true, false},
		{"success", 2, password, 200, true, true},
		{"already spoilered", 4, password, 200, true, true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			data := spoilerRequest{
				ID:       c.id,
				Password: c.password,
			}
			rec, req := newJSONPair(t, "/spoiler", data)
			router.ServeHTTP(rec, req)

			assertCode(t, rec, c.code)

			post, err := db.GetPost(c.id)
			if err != nil {
				t.Fatal(err)
			}
			if c.hasImage && post.Image.Spoiler != c.spoilered {
				assertRepLogContains(t, 1, "11"+strconv.FormatUint(c.id, 10))
				t.Errorf(
					"spoiler mismatch: expected %v; got %v",
					c.spoilered,
					post.Image.Spoiler,
				)
			}
		})
	}
}

func writeSampleImage(t *testing.T) {
	if err := db.WriteImage(assets.StdJPEG.ImageCommon); err != nil {
		t.Fatal(err)
	}
}

func assertRepLogContains(t *testing.T, id uint64, msg string) {
	res, err := db.GetLog(id, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	contains := false
	for _, r := range res {
		if string(r) == msg {
			contains = true
			break
		}
	}
	if !contains {
		t.Errorf(`log does not contain message "%s"`, msg)
	}
}
