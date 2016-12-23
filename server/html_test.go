package server

import (
	"testing"

	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

func TestThreadHTML(t *testing.T) {
	cache.Clear()
	assertTableClear(t, "threads", "posts")
	assertInsert(t, "threads", common.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	assertInsert(t, "posts", common.DatabasePost{
		StandalonePost: common.StandalonePost{
			OP:    1,
			Board: "a",
			Post: common.Post{
				ID: 1,
			},
		},
	})
	setBoards(t, "a")
	(*config.Get()).DefaultLang = "en_GB"

	cases := [...]struct {
		name, url string
		code      int
	}{
		{"unparsable thread number", "/a/www", 404},
		{"nonexistent thread", "/a/22", 404},
		{"thread exists", "/a/1", 200},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair(c.url)
			router.ServeHTTP(rec, req)
			assertCode(t, rec, c.code)
		})
	}
}

func TestBoardHTML(t *testing.T) {
	cache.Clear()
	setupPosts(t)
	setBoards(t, "a")
	(*config.Get()).DefaultLang = "en_GB"

	cases := [...]struct {
		name, url string
		code      int
	}{
		{"/all/ board", "/all/", 200},
		{"regular board", "/a/", 200},
		{"without index template", "/a/?noIndex=true", 200},
		{"non-existent board", "/b/", 404},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair(c.url)
			router.ServeHTTP(rec, req)
			assertCode(t, rec, c.code)
		})
	}
}

func TestOwnedBoardSelection(t *testing.T) {
	config.ClearBoards()
	conf := [...]config.BoardConfigs{
		{
			ID: "a",
			Staff: map[string][]string{
				"owners": {"foo", "admin"},
			},
		},
		{
			ID: "c",
			Staff: map[string][]string{
				"owners": {"admin"},
			},
		},
	}
	for _, c := range conf {
		if _, err := config.SetBoardConfigs(c); err != nil {
			t.Fatal(err)
		}
	}
	(*config.Get()).DefaultLang = "en_GB"

	cases := [...]struct {
		name, id string
	}{
		{"no owned boards", "bar"},
		{"one owned board", "foo"},
		{"multiple owned boards", "admin"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair("/forms/ownedBoards/" + c.id)
			router.ServeHTTP(rec, req)
			assertCode(t, rec, 200)
		})
	}
}

func TestBoardConfigurationForm(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)
	config.ClearBoards()

	conf := config.BoardConfigs{
		ID: "a",
		Staff: map[string][]string{
			"owners": {"user1"},
		},
	}
	_, err := config.SetBoardConfigs(conf)
	if err != nil {
		t.Fatal(err)
	}

	(*config.Get()).DefaultLang = "en_GB"

	rec, req := newJSONPair(t, "/forms/configureBoard", boardConfigRequest{
		ID:           "a",
		SessionCreds: sampleLoginCreds,
	})
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)
}

func TestStaticTemplates(t *testing.T) {
	(*config.Get()).DefaultLang = "en_GB"

	cases := [...]struct {
		name, url string
	}{
		{"create board", "/forms/createBoard"},
		{"board navigation panel", "/forms/boardNavigation"},
		{"change password", "/forms/changePassword"},
		{"captcha confirmation", "/forms/captcha"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair(c.url)
			router.ServeHTTP(rec, req)
			assertCode(t, rec, 200)
		})
	}
}

func TestServerConfigurationForm(t *testing.T) {
	assertTableClear(t, "accounts")
	writeAdminAccount(t)
	(*config.Get()).DefaultLang = "en_GB"

	rec, req := newJSONPair(t, "/forms/configureServer", adminLoginCreds)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)
}
