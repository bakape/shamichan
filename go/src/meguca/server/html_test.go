package server

import (
	"database/sql"
	"testing"

	"meguca/cache"
	"meguca/config"
	"meguca/db"
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
		err := db.InTransaction(func(tx *sql.Tx) error {
			return db.WriteBoard(tx, db.BoardConfigs{
				BoardConfigs: config.BoardConfigs{
					ID:        b,
					Eightball: []string{"yes"},
				},
			})
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
	err := db.InTransaction(func(tx *sql.Tx) error {
		for _, s := range staff {
			err := db.WriteStaff(tx, s.id, map[string][]string{
				"owners": s.owners,
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		return nil
	})
	if err != nil {
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

			rec, req := newPair("/html/owned-boards/" + c.id)
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

	err := db.InTransaction(func(tx *sql.Tx) error {
		return db.WriteStaff(tx, "a", map[string][]string{
			"owners": {"user1"},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	rec, req := newJSONPair(t, "/html/configure-board/a", nil)
	setLoginCookies(req, sampleLoginCreds)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)
}

func TestStaticTemplates(t *testing.T) {
	(*config.Get()).DefaultLang = "en_GB"

	cases := [...]struct {
		name, url string
	}{
		{"create board", "/html/create-board"},
		{"board navigation panel", "/html/board-navigation"},
		{"change password", "/html/change-password"},
		{"captcha confirmation", "/html/captcha"},
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

	rec, req := newJSONPair(t, "/html/configure-server", nil)
	setLoginCookies(req, adminLoginCreds)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)
}
