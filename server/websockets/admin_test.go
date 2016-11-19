package websockets

import (
	"testing"

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
