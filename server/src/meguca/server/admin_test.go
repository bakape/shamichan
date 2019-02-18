package server

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	. "meguca/test"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var adminLoginCreds = auth.SessionCreds{
	UserID:  "admin",
	Session: genSession(),
}

func newJSONPair(t *testing.T, url string, data interface{}) (
	*httptest.ResponseRecorder, *http.Request,
) {
	t.Helper()

	body := encodeBody(t, data)
	return httptest.NewRecorder(), httptest.NewRequest("POST", url, body)
}

func encodeBody(t *testing.T, data interface{}) io.Reader {
	t.Helper()
	return bytes.NewReader(marshalJSON(t, data))
}

func TestNotBoardOwner(t *testing.T) {
	assertTableClear(t, "accounts", "boards")
	writeSampleBoard(t)
	writeSampleUser(t)

	paths := [...]string{
		"/api/configure-board/a",
		"/html/configure-board/a",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, p, map[string]string{})
			router.ServeHTTP(rec, req)
			assertError(t, rec, 403, errAccessDenied)
		})
	}
}

func TestServePrivateBoardConfigs(t *testing.T) {
	assertTableClear(t, "boards", "accounts")

	config.ClearBoards()
	conf := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"a", "b", "c"},
		},
	}
	_, err := config.SetBoardConfigs(conf.BoardConfigs)
	if err != nil {
		t.Fatal(err)
	}
	err = db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteBoard(tx, conf)
	})
	if err != nil {
		t.Fatal(err)
	}
	writeSampleUser(t)
	writeSampleBoardOwner(t)

	rec, req := newJSONPair(t, "/api/board-config/a", nil)
	setLoginCookies(req, sampleLoginCreds)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)
	assertBody(t, rec, string(marshalJSON(t, conf.BoardConfigs)))
}

func TestBoardConfiguration(t *testing.T) {
	assertTableClear(t, "accounts", "boards")
	(*config.Get()).Captcha = false

	const board = "a"
	conf := config.BoardConfigs{
		ID:        board,
		Eightball: []string{},
		BoardPublic: config.BoardPublic{
			ForcedAnon: true,
			DefaultCSS: "moe",
		},
	}
	init := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			BoardPublic: config.BoardPublic{
				DefaultCSS: "moe",
			},
			ID:        board,
			Eightball: []string{},
		},
	}
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteBoard(tx, init)
	})
	if err != nil {
		t.Fatal(err)
	}

	writeSampleUser(t)
	writeSampleBoardOwner(t)

	data := conf
	rec, req := newJSONPair(t, "/api/configure-board/a", data)
	setLoginCookies(req, sampleLoginCreds)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)

	res, err := db.GetBoardConfigs(board)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, res, conf)
}

func TestValidateBoardConfigs(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name string
		config.BoardConfigs
		err error
	}{
		{
			"all is well",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					DefaultCSS: "moe",
				},
			},
			nil,
		},
		{
			"too many eightball answers",
			config.BoardConfigs{
				Eightball: make([]string, maxEightballLen+1),
			},
			errTooManyAnswers,
		},
		{
			"compound eightball length too big",
			config.BoardConfigs{
				Eightball: []string{GenString(maxEightballLen + 1)},
			},
			errEightballTooLong,
		},
		{
			"notice too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Notice: GenString(common.MaxLenNotice + 1),
				},
			},
			errNoticeTooLong,
		},
		{
			"rules too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Rules: GenString(common.MaxLenRules + 1),
				},
			},
			errRulesTooLong,
		},
		{
			"title too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Title: GenString(common.MaxLenBoardTitle + 1),
				},
			},
			errTitleTooLong,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			err := validateBoardConfigs(rec, c.BoardConfigs)
			if err != c.err {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestValidateBoardCreation(t *testing.T) {
	assertTableClear(t, "boards", "accounts")
	writeSampleBoard(t)
	writeSampleUser(t)
	(*config.Get()).Captcha = false

	cases := [...]struct {
		name, id, title string
		err             error
	}{
		{
			name:  "board name too long",
			id:    GenString(common.MaxLenBoardID + 1),
			title: "foo",
			err:   errInvalidBoardName,
		},
		{
			name:  "empty board name",
			id:    "",
			title: "foo",
			err:   errInvalidBoardName,
		},
		{
			name:  "invalid chars in board name",
			id:    ":^)",
			title: "foo",
			err:   errInvalidBoardName,
		},
		{
			name:  "title too long",
			id:    "b",
			title: GenString(101),
			err:   errTitleTooLong,
		},
		{
			name:  "board name taken",
			id:    "a",
			title: "foo",
			err:   errBoardNameTaken,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			msg := boardCreationRequest{
				ID:    c.id,
				Title: c.title,
			}
			rec, req := newJSONPair(t, "/api/create-board", msg)
			setLoginCookies(req, sampleLoginCreds)
			router.ServeHTTP(rec, req)

			assertCode(t, rec, 400)
			assertBody(t, rec, fmt.Sprintf("400 %s\n", c.err))
		})
	}
}

func writeSampleBoard(t testing.TB) {
	t.Helper()

	b := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteBoard(tx, b)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.SetBoardConfigs(b.BoardConfigs); err != nil {
		t.Fatal(err)
	}
}

func writeSampleBoardOwner(t *testing.T) {
	t.Helper()
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteStaff(tx, "a", map[string][]string{
			"owners": {"user1"},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBoardCreation(t *testing.T) {
	assertTableClear(t, "boards", "accounts")
	writeSampleUser(t)
	(*config.Get()).Captcha = false

	const (
		id    = "a"
		title = "/a/ - Animu & Mango"
	)

	msg := boardCreationRequest{
		ID:    id,
		Title: title,
	}
	rec, req := newJSONPair(t, "/api/create-board", msg)
	setLoginCookies(req, sampleLoginCreds)
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)

	board, err := db.GetBoardConfigs(id)
	if err != nil {
		t.Fatal(err)
	}

	std := config.BoardConfigs{
		ID: id,
		BoardPublic: config.BoardPublic{
			Title: title,
		},
		Eightball: config.EightballDefaults,
	}
	AssertDeepEquals(t, board, std)
}

func TestServePrivateServerConfigs(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)
	writeAdminAccount(t)
	if err := config.Set(config.Defaults); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name string
		auth.SessionCreds
		code int
		err  error
	}{
		{
			name:         "not admin",
			SessionCreds: sampleLoginCreds,
			code:         403,
			err:          errAccessDenied,
		},
		{
			name:         "admin",
			SessionCreds: adminLoginCreds,
			code:         200,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/api/config", nil)
			setLoginCookies(req, c.SessionCreds)
			router.ServeHTTP(rec, req)

			assertCode(t, rec, c.code)
			if c.err != nil {
				assertBody(t, rec, fmt.Sprintf("%d %s\n", c.code, c.err))
			}
		})
	}
}

func writeAdminAccount(t *testing.T) {
	t.Helper()

	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.CreateAdminAccount(tx)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.WriteLoginSession("admin", adminLoginCreds.Session)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerConfigSetting(t *testing.T) {
	assertTableClear(t, "accounts")
	if err := db.WriteConfigs(config.Defaults); err != nil {
		t.Fatal(err)
	}
	writeAdminAccount(t)

	msg := config.Defaults
	msg.DefaultCSS = "ashita"
	rec, req := newJSONPair(t, "/api/configure-server", msg)
	setLoginCookies(req, adminLoginCreds)
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)

	conf, err := db.GetConfigs()
	if err != nil {
		t.Fatal(err)
	}
	std := config.Defaults
	std.DefaultCSS = "ashita"
	AssertDeepEquals(t, conf, std)
}

func TestDeleteBoard(t *testing.T) {
	assertTableClear(t, "accounts", "boards")
	writeSampleUser(t)
	writeSampleBoard(t)
	writeSampleBoardOwner(t)
	writeAllBoard(t)
	(*config.Get()).Captcha = false

	rec, req := newJSONPair(t, "/api/delete-board", boardActionRequest{
		Board: "a",
	})
	setLoginCookies(req, sampleLoginCreds)
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)
}

// Restore all board to enable global logging
func writeAllBoard(t *testing.T) {
	t.Helper()

	err := db.InTransaction(false, func(tx *sql.Tx) (err error) {
		err = db.WriteBoard(tx, db.BoardConfigs{
			BoardConfigs: config.AllBoardConfigs.BoardConfigs,
			Created:      time.Now().UTC(),
		})
		if err != nil {
			return
		}
		return db.CreateSystemAccount(tx)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeletePost(t *testing.T) {
	assertTableClear(t, "accounts", "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	writeSampleUser(t)
	writeSampleBoardOwner(t)

	cConfigs := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "c",
			Eightball: []string{"yes"},
		},
	}
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteBoard(tx, cConfigs)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := config.SetBoardConfigs(cConfigs.BoardConfigs); err != nil {
		t.Fatal(err)
	}

	thread := db.Thread{
		ID:    3,
		Board: "c",
	}
	op := db.Post{
		StandalonePost: common.StandalonePost{
			Board: "c",
			Post: common.Post{
				ID: 3,
			},
			OP: 3,
		},
	}
	err = db.WriteThread(thread, op)
	if err != nil {
		t.Fatal(err)
	}

	posts := [...]db.Post{
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 2,
				},
				OP: 1,
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 4,
				},
				OP: 1,
			},
		},
	}
	err = db.InTransaction(false, func(tx *sql.Tx) (err error) {
		for _, p := range posts {
			err = db.WritePost(tx, p)
			if err != nil {
				return
			}
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	data := []uint64{2, 4}
	const url = "/api/delete-post"
	rec, req := newJSONPair(t, url, data)
	setLoginCookies(req, sampleLoginCreds)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)

	data = []uint64{3}
	rec, req = newJSONPair(t, url, data)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 403)

	cases := [...]struct {
		name    string
		id      uint64
		deleted bool
	}{
		{"from target board", 2, true},
		{"from target board", 4, true},
		{"different board", 3, false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			post, err := db.GetPost(c.id)
			switch {
			case err != nil:
				t.Fatal(err)
			case post.IsDeleted() != c.deleted:
				LogUnexpected(t, post.IsDeleted(), c.deleted)
			}
		})
	}
}

func writeSampleThread(t *testing.T) {
	t.Helper()

	thread := db.Thread{
		ID:        1,
		Board:     "a",
		ReplyTime: 11,
	}
	op := db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Time: 345351,
			},
			OP:    1,
			Board: "a",
		},
	}
	err := db.WriteThread(thread, op)
	if err != nil {
		t.Fatal(err)
	}
}
