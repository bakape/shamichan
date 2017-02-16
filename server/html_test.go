package server

import (
	"testing"

	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
)

func TestThreadHTML(t *testing.T) {
	cache.Clear()
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
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
		{"without index template", "/a/?minimal=true", 200},
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
	assertTableClear(t, "boards", "accounts")
	config.ClearBoards()
	(*config.Get()).DefaultLang = "en_GB"
	writeAdminAccount(t)
	writeSampleUser(t)

	for _, b := range [...]string{"a", "c"} {
		err := db.WriteBoard(nil, db.BoardConfigs{
			BoardConfigs: config.BoardConfigs{
				ID:        b,
				Eightball: []string{"yes"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		conf := config.BoardConfigs{
			ID: b,
		}
		if _, err := config.SetBoardConfigs(conf); err != nil {
			t.Fatal(err)
		}
	}

	staff := [...]struct {
		id     string
		owners []string
	}{
		{
			"a",
			[]string{"user1", "admin"},
		},
		{
			"c",
			[]string{"admin"},
		},
	}
	tx, err := db.StartTransaction()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range staff {
		err := db.WriteStaff(tx, s.id, map[string][]string{
			"owners": s.owners,
		})
		if err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, id string
	}{
		{"no owned boards", "bar"},
		{"one owned board", "user1"},
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
	config.ClearBoards()
	(*config.Get()).DefaultLang = "en_GB"
	assertTableClear(t, "accounts", "boards")
	writeSampleBoard(t)
	writeSampleUser(t)

	tx, err := db.StartTransaction()
	if err != nil {
		t.Fatal(err)
	}
	err = db.WriteStaff(tx, "a", map[string][]string{
		"owners": {"user1"},
	})
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

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
