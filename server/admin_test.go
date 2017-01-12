package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	"github.com/dancannon/gorethink"
)

var adminLoginCreds = auth.SessionCreds{
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
		ID:           "a",
		SessionCreds: sampleLoginCreds,
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
		SessionCreds: sampleLoginCreds,
		BoardConfigs: conf,
	}
	rec, req := newJSONPair(t, "/admin/configureBoard", data)
	router.ServeHTTP(rec, req)

	var res config.BoardConfigs
	if err := db.One(gorethink.Table("boards").Get(board), &res); err != nil {
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
				Eightball: []string{GenString(maxEightballLen + 1)},
			},
			errEightballTooLong,
		},
		{
			"notice too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Notice: GenString(maxNoticeLen + 1),
				},
			},
			errNoticeTooLong,
		},
		{
			"rules too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Rules: GenString(maxRulesLen + 1),
				},
			},
			errRulesTooLong,
		},
		{
			"title too long",
			config.BoardConfigs{
				BoardPublic: config.BoardPublic{
					Title: GenString(maxTitleLen + 1),
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
				Name:         c.id,
				Title:        c.title,
				SessionCreds: sampleLoginCreds,
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
		Name:         id,
		Title:        title,
		SessionCreds: sampleLoginCreds,
	}
	rec, req := newJSONPair(t, "/admin/createBoard", msg)
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)

	var board config.DatabaseBoardConfigs
	if err := db.One(gorethink.Table("boards").Get(id), &board); err != nil {
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

			rec, req := newJSONPair(t, "/admin/config", c.SessionCreds)
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
	assertTableClear(t, "accounts")
	assertInsert(t, "main", db.ConfigDocument{
		Document: db.Document{
			ID: "config",
		},
		Configs: config.Defaults,
	})
	writeAdminAccount(t)

	msg := configSettingRequest{
		SessionCreds: adminLoginCreds,
		Configs:      config.Defaults,
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

func TestDeleteBoard(t *testing.T) {
	assertTableClear(t, "accounts", "threads", "posts", "boards")
	writeSampleUser(t)
	assertInsert(t, "boards", config.DatabaseBoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID: "a",
			Staff: map[string][]string{
				"owners": {sampleLoginCreds.UserID},
			},
		},
	})

	rec, req := newJSONPair(t, "/admin/deleteBoard", boardDeletionRequest{
		ID:           "a",
		SessionCreds: sampleLoginCreds,
	})
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)
	assertDeleted(t, gorethink.Table("boards").Get("a"), true)
}

func assertDeleted(t *testing.T, q gorethink.Term, del bool) {
	var deleted bool
	if err := db.One(q.Eq(nil), &deleted); err != nil {
		t.Fatal(err)
	}
	if deleted != del {
		LogUnexpected(t, del, deleted)
	}
}

func TestDeletePost(t *testing.T) {
	assertTableClear(t, "main", "accounts", "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 1,
				},
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Board: "c",
				Post: common.Post{
					ID: 2,
				},
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 3,
				},
			},
		},
	})
	writeSampleUser(t)

	rec, req := newJSONPair(t, "/admin/deletePost", postActionRequest{
		IDs:          []uint64{1, 2, 3},
		Board:        "a",
		SessionCreds: sampleLoginCreds,
	})
	router.ServeHTTP(rec, req)

	assertCode(t, rec, 200)

	cases := [...]struct {
		name    string
		id      uint64
		deleted bool
	}{
		{"from target board", 1, true},
		{"from target board", 3, true},
		{"different board", 2, false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var deleted bool
			q := db.FindPost(c.id).Field("deleted").Default(false)
			if err := db.One(q, &deleted); err != nil {
				t.Fatal(err)
			}
			if deleted != c.deleted {
				LogUnexpected(t, deleted, c.deleted)
			}

			if !c.deleted {
				return
			}
			msg, err := common.EncodeMessage(common.MessageDeletePost, c.id)
			if err != nil {
				t.Fatal(err)
			}
			var contains bool
			q = db.FindPost(c.id).Field("log").Contains(msg)
			if err := db.One(q, &contains); err != nil {
				t.Fatal(err)
			}
			if !contains {
				t.Errorf("log message not written")
			}
		})
	}
}
