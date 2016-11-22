package server

import (
	"strconv"
	"testing"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/dancannon/gorethink"
)

func TestSpoilerImage(t *testing.T) {
	assertTableClear(t, "posts")

	const password = "123"
	hash, err := auth.BcryptHash(password, 6)
	if err != nil {
		t.Fatal(err)
	}

	assertInsert(t, "posts", []common.DatabasePost{
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 1,
					Image: &common.Image{
						ImageCommon: common.ImageCommon{
							SHA1: "123",
						},
					},
				},
			},
		},
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 2,
				},
			},
		},
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 3,
					Image: &common.Image{
						ImageCommon: common.ImageCommon{
							SHA1: "123",
						},
						Spoiler: true,
					},
				},
			},
		},
		{
			Password: hash,
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 4,
					Image: &common.Image{
						ImageCommon: common.ImageCommon{
							SHA1: "123",
						},
					},
				},
			},
		},
	})

	cases := [...]struct {
		name      string
		id        int64
		password  string
		code      int
		spoilered bool
	}{
		{"no image", 2, password, 400, false},
		{"wrong password", 4, "122", 403, false},
		{"success", 1, password, 200, true},
		{"already spoilered", 3, password, 200, true},
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

			var spoilered bool
			msg := []byte("11" + strconv.Itoa(int(c.id)))
			post := db.FindPost(c.id)
			q := gorethink.And(
				post.Field("log").Contains(msg),
				post.Field("image").Field("spoiler"),
			)
			if err := db.One(q, &spoilered); err != nil {
				t.Fatal(err)
			}
			if spoilered != spoilered {
				t.Errorf(
					"spoiler mismatch: expected %v; got %v",
					c.spoilered,
					spoilered,
				)
			}
		})
	}
}
