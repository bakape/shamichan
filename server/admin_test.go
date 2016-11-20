package server

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	r "github.com/dancannon/gorethink"
)

var adminLoginCreds = loginCredentials{
	UserID:  "admin",
	Session: genSession(),
}

func newJSONPair(t *testing.T, url string, data interface{}) (
	*httptest.ResponseRecorder, *http.Request,
) {
	body := encodeBody(t, data)
	return httptest.NewRecorder(), httptest.NewRequest("POST", url, body)
}

func encodeBody(t *testing.T, data interface{}) io.Reader {
	return bytes.NewReader(marshalJSON(t, data))
}

func TestNotBoardOwner(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)

	fns := [...]http.HandlerFunc{configureBoard, servePrivateBoardConfigs}

	for i := range fns {
		fn := fns[i]
		t.Run("", func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/", sampleLoginCreds)
			fn(rec, req)
			assertCode(t, rec, 403)
			assertBody(t, rec, "403 Not board owner\n")
		})
	}
}

func TestServePrivateBoardConfigs(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)
	config.ClearBoards()

	conf := config.BoardConfigs{
		ID:        "a",
		Eightball: []string{"a", "b", "c"},
		BoardPublic: config.BoardPublic{
			Banners: []string{},
		},
		Staff: map[string][]string{
			"owners": {"user1"},
		},
	}
	_, err := config.SetBoardConfigs(conf)
	if err != nil {
		t.Fatal(err)
	}

	rec, req := newJSONPair(t, "/admin/boardConfig", boardConfigRequest{
		ID:               "a",
		loginCredentials: sampleLoginCreds,
	})
	router.ServeHTTP(rec, req)
	assertBody(t, rec, string(marshalJSON(t, conf)))
}

func TestBoardConfiguration(t *testing.T) {
	assertTableClear(t, "accounts", "boards")

	const (
		id    = "user1"
		board = "a"
	)
	staff := map[string][]string{
		"owners": {id},
	}
	conf := config.BoardConfigs{
		ID: board,
		BoardPublic: config.BoardPublic{
			PostParseConfigs: config.PostParseConfigs{
				ForcedAnon: true,
			},
			Banners: []string{},
			Spoiler: "default.jpg",
		},
		Eightball: []string{},
		Staff:     staff,
	}
	init := config.BoardConfigs{
		ID: board,
		BoardPublic: config.BoardPublic{
			Banners: []string{},
		},
		Eightball: []string{},
		Staff:     staff,
	}
	assertInsert(t, "boards", init)

	writeSampleUser(t)

	data := boardConfigSettingRequest{
		loginCredentials: sampleLoginCreds,
		BoardConfigs:     conf,
	}
	rec, req := newJSONPair(t, "/admin/configureBoard", data)
	router.ServeHTTP(rec, req)

	var res config.BoardConfigs
	if err := db.One(r.Table("boards").Get(board), &res); err != nil {
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
			config.BoardConfigs{},
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
			"compound eightball length to big",
			config.BoardConfigs{
				Eightball: []string{genString(maxEightballLen + 1)},
			},
			errEightballTooLong,
		},
		{
			"notice too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Notice: genString(maxNoticeLen + 1),
				},
			},
			errNoticeTooLong,
		},
		{
			"rules too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Rules: genString(maxRulesLen + 1),
				},
			},
			errRulesTooLong,
		},
		{
			"title too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Title: genString(maxTitleLen + 1),
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
			if b := validateBoardConfigs(rec, c.BoardConfigs); b != (c.err == nil) {
				t.Fatal("unexpected result")
			}
			if c.err != nil {
				assertCode(t, rec, 400)
				assertBody(t, rec, fmt.Sprintf("400 %s\n", c.err))
			}
		})
	}
}

func genString(len int) string {
	buf := make([]byte, len)
	for i := 0; i < len; i++ {
		buf[i] = byte(rand.Intn(128))
	}
	return string(buf)
}

func TestValidateBoardCreation(t *testing.T) {
	assertTableClear(t, "boards", "accounts")
	assertInsert(t, "boards", db.Document{ID: "a"})
	writeSampleUser(t)

	cases := [...]struct {
		name, id, title string
		err             error
	}{
		{
			name:  "board name too long",
			id:    "abcd",
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
			name:  "reserved key 'id' as board name",
			id:    "id",
			title: "foo",
			err:   errInvalidBoardName,
		},
		{
			name:  "title too long",
			id:    "b",
			title: genString(101),
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
				Name:             c.id,
				Title:            c.title,
				loginCredentials: sampleLoginCreds,
			}
			rec, req := newJSONPair(t, "/admin/createBoard", msg)
			router.ServeHTTP(rec, req)

			assertCode(t, rec, 400)
			assertBody(t, rec, fmt.Sprintf("400 %s\n", c.err))
		})
	}
}

func TestBoardCreation(t *testing.T) {
	assertTableClear(t, "boards", "accounts")
	writeSampleUser(t)

	const (
		id     = "a"
		userID = "user1"
		title  = "/a/ - Animu & Mango"
	)

	msg := boardCreationRequest{
		Name:             id,
		Title:            title,
		loginCredentials: sampleLoginCreds,
	}
	rec, req := newJSONPair(t, "/admin/createBoard", msg)
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)

	var board config.DatabaseBoardConfigs
	if err := db.One(r.Table("boards").Get(id), &board); err != nil {
		t.Fatal(err)
	}

	std := config.DatabaseBoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID: id,
			BoardPublic: config.BoardPublic{
				Spoiler: "default.jpg",
				Title:   title,
				Banners: []string{},
			},
			Eightball: config.EightballDefaults,
			Staff: map[string][]string{
				"owners": []string{userID},
			},
		},
	}

	c := board.Created
	if !c.Before(time.Now()) || c.Unix() == 0 {
		t.Errorf("invalid board creation time: %#v", board.Created)
	}
	AssertDeepEquals(t, board.BoardConfigs, std.BoardConfigs)
}

func TestServePrivateServerConfigs(t *testing.T) {
	assertTableClear(t, "accounts")
	writeSampleUser(t)
	writeAdminAccount(t)
	config.Set(config.Defaults)

	cases := [...]struct {
		name string
		loginCredentials
		code int
		err  error
	}{
		{
			name:             "not admin",
			loginCredentials: sampleLoginCreds,
			code:             403,
			err:              errAccessDenied,
		},
		{
			name:             "admin",
			loginCredentials: adminLoginCreds,
			code:             200,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newJSONPair(t, "/admin/config", c.loginCredentials)
			router.ServeHTTP(rec, req)

			assertCode(t, rec, c.code)
			if c.err != nil {
				assertBody(t, rec, fmt.Sprintf("%d %s\n", c.code, c.err))
			}
		})
	}
}

func writeAdminAccount(t *testing.T) {
	assertInsert(t, "accounts", auth.User{
		ID: adminLoginCreds.UserID,
		Sessions: []auth.Session{
			{
				Token:   adminLoginCreds.Session,
				Expires: time.Now().Add(time.Hour),
			},
		},
	})
}

func TestServerConfigSetting(t *testing.T) {
	assertTableClear(t, "main", "accounts")
	assertInsert(t, "main", db.ConfigDocument{
		Document: db.Document{
			ID: "config",
		},
		Configs: config.Defaults,
	})
	writeAdminAccount(t)

	msg := configSettingRequest{
		loginCredentials: adminLoginCreds,
		Configs:          config.Defaults,
	}
	msg.DefaultCSS = "ashita"
	rec, req := newJSONPair(t, "/admin/configureServer", msg)
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)

	var conf config.Configs
	if err := db.One(db.GetMain("config"), &conf); err != nil {
		t.Fatal(err)
	}
	std := config.Defaults
	std.DefaultCSS = "ashita"
	AssertDeepEquals(t, conf, std)
}
