package websockets

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/db"
	. "github.com/Chiiruno/meguca/test"
	"github.com/Chiiruno/meguca/test/test_db"
	"github.com/Chiiruno/meguca/websockets/feeds"
)

var (
	// JPEG sample image standard struct
	stdJPEG = common.ImageCommon{
		SHA1:     "012a2f912c9ee93ceb0ccb8684a29ec571990a94",
		FileType: 0,
		Dims:     [4]uint16{1, 1, 1, 1},
		MD5:      "YOQQklgfezKbBXuEAsqopw",
		Size:     300792,
	}
)

func TestInsertThread(t *testing.T) {
	test_db.ClearTables(t, "boards", "images")

	conf := [...]db.BoardConfigs{
		{
			BoardConfigs: config.BoardConfigs{
				ID: "c",
			},
		},
		{
			BoardConfigs: config.BoardConfigs{
				ID: "r",
				BoardPublic: config.BoardPublic{
					ReadOnly: true,
				},
			},
		},
		{
			BoardConfigs: config.BoardConfigs{
				ID: "a",
				BoardPublic: config.BoardPublic{
					TextOnly: true,
				},
			},
		},
	}
	for _, c := range conf {
		c.Eightball = []string{"Yes"}
		if _, err := config.SetBoardConfigs(c.BoardConfigs); err != nil {
			t.Fatal(err)
		}
		err := db.InTransaction(false, func(tx *sql.Tx) error {
			return db.WriteBoard(tx, c)
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	cases := [...]struct {
		name, board string
		err         error
	}{
		{"invalid board", "all", common.ErrInvalidBoard("all")},
		{"invalid board", "x", common.ErrInvalidBoard("x")},
		{"read-only board", "r", errReadOnly},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := ThreadCreationRequest{
				Board: c.board,
			}
			_, err := CreateThread(req, "")
			AssertEquals(t, c.err, err)
		})
	}

	t.Run("with image", testCreateThread)
	t.Run("text only board", testCreateThreadTextOnly)
}

func testCreateThread(t *testing.T) {
	writeSampleImage(t)
	var token string
	err := db.InTransaction(false, func(tx *sql.Tx) (err error) {
		token, err = db.NewImageToken(tx, stdJPEG.SHA1)
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	std := common.Thread{
		Board:      "c",
		Subject:    "subject",
		ImageCount: 1,
		PostCount:  1,
		Post: common.Post{
			Name: "name",
			Image: &common.Image{
				Spoiler:     true,
				ImageCommon: stdJPEG,
				Name:        "foo",
			},
		},
		Posts: []common.Post{},
	}

	req := ThreadCreationRequest{
		ReplyCreationRequest: ReplyCreationRequest{
			Name:     "name",
			Password: "123",
			Image: ImageRequest{
				Name:    "foo.jpeg",
				Token:   token,
				Spoiler: true,
			},
		},
		Subject: "subject",
		Board:   "c",
	}
	p, err := CreateThread(req, "::1")
	if err != nil {
		t.Fatal(err)
	}
	std.ID = p.ID

	thread, err := db.GetThread(p.ID, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Pointers have to be dereferenced to be asserted
	AssertEquals(t, *thread.Image, *std.Image)

	// Normalize timestamps and pointer fields
	then := thread.UpdateTime
	std.UpdateTime = then
	std.BumpTime = then
	std.Time = then
	std.Image = thread.Image

	AssertEquals(t, thread, std)
}

func testCreateThreadTextOnly(t *testing.T) {
	post, err := CreateThread(ThreadCreationRequest{
		ReplyCreationRequest: ReplyCreationRequest{
			Name:     "name",
			Password: "123",
		},
		Subject: "subject",
		Board:   "a",
	}, "::1")
	if err != nil {
		t.Fatal(err)
	}
	if post.Image != nil {
		t.Error("image inserted")
	}

	hasImage, err := db.HasImage(post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if hasImage {
		t.Error("image written to database")
	}
}

func setBoardConfigs(t testing.TB, textOnly bool) {
	t.Helper()

	config.ClearBoards()
	_, err := config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
		BoardPublic: config.BoardPublic{
			TextOnly: textOnly,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertIP(t *testing.T, id uint64, ip string) {
	t.Helper()

	res, err := db.GetIP(id)
	if err != nil {
		t.Fatal(err)
	}
	AssertEquals(t, res, ip)
}

func TestClosePreviousPostOnCreation(t *testing.T) {
	feeds.Clear()
	test_db.ClearTables(t, "boards")
	test_db.WriteSampleBoard(t)
	test_db.WriteSampleThread(t)
	writeSamplePost(t)
	if err := db.SetPostCounter(5); err != nil {
		t.Fatal(err)
	}
	setBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	registerClient(t, cl, 1, "a")
	cl.post = openPost{
		id:    2,
		op:    1,
		len:   3,
		board: "a",
		time:  time.Now().Unix(),
		body:  []byte("abc"),
	}
	data := marshalJSON(t, ReplyCreationRequest{
		Name:     "name",
		Body:     "foo",
		Password: "123",
	})

	if err := cl.insertPost(data); err != nil {
		t.Fatal(err)
	}

	assertMessage(t, wcl, `326`)
	assertPostClosed(t, 2)
}

func TestPostCreationValidations(t *testing.T) {
	setBoardConfigs(t, false)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	registerClient(t, cl, 1, "a")

	cases := [...]struct {
		testName, token, name string
	}{
		{"no token", "", "abc"},
		{"no image name", "abc", ""},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			req := ReplyCreationRequest{
				Image: ImageRequest{
					Name:  c.name,
					Token: c.token,
				},
			}
			err := cl.insertPost(marshalJSON(t, req))
			if err != errNoTextOrImage {
				UnexpectedError(t, err)
			}
		})
	}
}

func TestPostCreation(t *testing.T) {
	feeds.Clear()
	prepareForPostCreation(t)
	setBoardConfigs(t, false)
	writeSampleImage(t)
	var token string
	err := db.InTransaction(false, func(tx *sql.Tx) (err error) {
		token, err = db.NewImageToken(tx, stdJPEG.SHA1)
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.ip = "::1"
	registerClient(t, cl, 1, "a")
	defer cl.Close(nil)

	req := ReplyCreationRequest{
		Open:     true,
		Body:     "Δ",
		Password: "123",
		Image: ImageRequest{
			Name:    "foo.jpeg",
			Token:   token,
			Spoiler: true,
		},
	}

	err = cl.insertPost(marshalJSON(t, req))
	if err != nil {
		t.Fatal(err)
	}

	assertMessage(t, wcl, encodeMessageType(common.MessagePostID)+"6")

	stdPost := common.StandalonePost{
		Post: common.Post{
			Editing: true,
			ID:      6,
			Body:    "Δ",
			Image: &common.Image{
				Name:        "foo",
				Spoiler:     true,
				ImageCommon: stdJPEG,
			},
		},
		OP:    1,
		Board: "a",
	}

	post, err := db.GetPost(6)
	if err != nil {
		t.Fatal(err)
	}
	AssertEquals(t, *post.Image, *stdPost.Image)
	stdPost.Image = post.Image
	stdPost.Time = post.Time
	AssertEquals(t, post, stdPost)

	assertIP(t, 6, "::1")

	thread, err := db.GetThread(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	AssertEquals(t, thread.PostCount, uint32(2))
	AssertEquals(t, thread.ImageCount, uint32(1))

	AssertEquals(t, cl.post, openPost{
		id:          6,
		op:          1,
		time:        stdPost.Time,
		board:       "a",
		len:         1,
		hasImage:    true,
		isSpoilered: true,
		body:        []byte("Δ"),
	})
}

func registerClient(t testing.TB, cl *Client, id uint64, board string) {
	t.Helper()

	var err error
	cl.feed, err = feeds.SyncClient(cl, id, board)
	if err != nil {
		t.Fatal(err)
	}
}

func encodeMessageType(typ common.MessageType) string {
	return strconv.Itoa(int(typ))
}

func prepareForPostCreation(t testing.TB) {
	t.Helper()

	test_db.ClearTables(t, "boards", "images", "bans")
	test_db.WriteSampleBoard(t)
	test_db.WriteSampleThread(t)
	if err := db.SetPostCounter(5); err != nil {
		t.Fatal(err)
	}
}

func TestTextOnlyPostCreation(t *testing.T) {
	feeds.Clear()
	prepareForPostCreation(t)
	setBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	registerClient(t, cl, 1, "a")
	defer cl.Close(nil)

	req := ReplyCreationRequest{
		Body:     "a",
		Password: "123",
	}

	if err := cl.insertPost(marshalJSON(t, req)); err != nil {
		t.Fatal(err)
	}

	// Assert no image in post
	hasImage, err := db.HasImage(6)
	if err != nil {
		t.Fatal(err)
	}
	if hasImage {
		t.Error("DB post has image")
	}
	if cl.post.hasImage {
		t.Error("openPost has image")
	}
}

func BenchmarkPostCreation(b *testing.B) {
	feeds.Clear()
	prepareForPostCreation(b)
	setBoardConfigs(b, true)

	sv := newWSServer(b)
	defer sv.Close()
	cl, _ := sv.NewClient()
	registerClient(b, cl, 1, "a")
	defer cl.Close(nil)

	req := ReplyCreationRequest{
		Body:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
		Password: "123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cl.insertPost(marshalJSON(b, req)); err != nil {
			b.Fatal(err)
		}
		if err := cl.closePost(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestPostCreationForcedAnon(t *testing.T) {
	feeds.Clear()
	prepareForPostCreation(t)
	config.ClearBoards()
	_, err := config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
		BoardPublic: config.BoardPublic{
			ForcedAnon: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	registerClient(t, cl, 1, "a")
	defer cl.Close(nil)

	req := ReplyCreationRequest{
		Body:     "a",
		Password: "123",
	}

	if err := cl.insertPost(marshalJSON(t, req)); err != nil {
		t.Fatal(err)
	}

	// Assert no name or trip in post
	post, err := db.GetPost(6)
	if err != nil {
		t.Fatal(err)
	}
	if post.Trip != "" || post.Name != "" {
		t.Fatal("not anonymous")
	}
}
