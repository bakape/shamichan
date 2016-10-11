package websockets

import (
	"testing"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
)

func TestNotAdmin(t *testing.T) {
	t.Parallel()

	cl := &Client{}
	cl.UserID = "foo"
	for _, fn := range []handler{configServer} {
		if err := fn(nil, cl); err != errAccessDenied {
			UnexpectedError(t, err)
		}
	}
}

func TestServerConfigRequest(t *testing.T) {
	config.Set(config.Defaults)

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.UserID = "admin"

	if err := configServer([]byte{}, cl); err != nil {
		t.Fatal(err)
	}

	msg, err := EncodeMessage(MessageConfigServer, config.Get())
	if err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, string(msg))
}

func TestServerConfigSetting(t *testing.T) {
	assertTableClear(t, "main")
	assertInsert(t, "main", db.ConfigDocument{
		Document: db.Document{
			ID: "config",
		},
		Configs: config.Defaults,
	})

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.UserID = "admin"

	req := config.Defaults
	req.Boards = []string{"fa"}
	req.DefaultCSS = "ashita"
	if err := configServer(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, "39true")

	var conf config.Configs
	if err := db.One(db.GetMain("config"), &conf); err != nil {
		t.Fatal(err)
	}
	std := config.Defaults
	std.DefaultCSS = "ashita"
	AssertDeepEquals(t, conf, std)
}

func TestValidateBoardCreation(t *testing.T) {
	assertTableClear(t, "boards")
	assertInsert(t, "boards", db.Document{ID: "a"})

	cases := [...]struct {
		name, boardName, title, response string
	}{
		{"board name too long", "abcd", "foo", "401"},
		{"empty board name", "", "foo", "401"},
		{"invalid chars in board name", ":^)", "foo", "401"},
		{"reserved key 'id' as board name", "id", "foo", "401"},
		{"title too long", "a", genString(101), "403"},
		{"board name taken", "a", "foo", "402"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := boardCreationRequest{
				Name:  c.boardName,
				Title: c.title,
			}
			assertLoggedInResponse(t, req, createBoard, "123", c.response)
		})
	}
}

func TestBoardCreation(t *testing.T) {
	assertTableClear(t, "main", "boards")

	const (
		id     = "a"
		userID = "123"
		title  = "/a/ - Animu & Mango"
	)
	assertInsert(t, "main", db.ConfigDocument{
		Document: db.Document{ID: "config"},
		Configs:  config.Defaults,
	})

	req := boardCreationRequest{
		Name:  id,
		Title: title,
	}
	assertLoggedInResponse(t, req, createBoard, userID, "400")

	var board config.DatabaseBoardConfigs
	if err := db.One(db.GetBoardConfig(id), &board); err != nil {
		t.Fatal(err)
	}
	std := config.DatabaseBoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        id,
			Spoiler:   "default.jpg",
			Title:     title,
			Eightball: config.EightballDefaults,
			Banners:   []string{},
			Staff: map[string][]string{
				"owners": []string{userID},
			},
		},
	}
	if !board.Created.Before(time.Now()) {
		t.Errorf("invalid board creation time: %#v", board.Created)
	}
	AssertDeepEquals(t, board.BoardConfigs, std.BoardConfigs)

	var boards []string
	err := db.All(db.GetMain("config").Field("boards"), &boards)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, boards, []string{"a"})
}
