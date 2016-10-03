package websockets

import (
	"bytes"
	"testing"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
)

var (
	// JPEG sample image standard struct
	stdJPEG = types.ImageCommon{
		SHA1:     "012a2f912c9ee93ceb0ccb8684a29ec571990a94",
		FileType: 0,
		Dims:     [4]uint16{1, 1, 1, 1},
		MD5:      "60e41092581f7b329b057b8402caa8a7",
		Size:     300792,
	}

	sampleImagelessThreadCreationRequest = threadCreationRequest{
		postCreationCommon: postCreationCommon{
			Name:     "name",
			Password: "123",
		},
		Subject: "subject",
		Board:   "a",
	}
)

func TestInsertThread(t *testing.T) {
	assertTableClear(t, "main", "posts", "threads", "boards", "images")
	populateMainTable(t)
	assertInsert(t, "boards", []config.BoardConfigs{
		{
			ID:               "c",
			PostParseConfigs: config.PostParseConfigs{},
		},
		{
			ID: "r",
			PostParseConfigs: config.PostParseConfigs{
				ReadOnly: true,
			},
		},
		{
			ID: "a",
			PostParseConfigs: config.PostParseConfigs{
				TextOnly: true,
			},
		},
	})
	(*config.Get()).Boards = []string{"a", "c", "r"}

	cases := [...]struct {
		name, board string
		err         error
	}{
		{"invalid board", "all", errInvalidBoard},
		{"invalid board", "x", errInvalidBoard},
		{"read-only board", "r", errReadOnly},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := threadCreationRequest{
				Board: c.board,
			}
			err := insertThread(marshalJSON(t, req), new(Client))
			if err != c.err {
				t.Fatalf("unexpected error %#v", err)
			}
		})
	}

	t.Run("with image", testCreateThread)
	t.Run("text only board", testCreateThreadTextOnly)
}

func testCreateThread(t *testing.T) {
	assertInsert(t, "images", stdJPEG)
	_, token, err := db.NewImageToken(stdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	Clients.Add(cl, SyncID{
		OP:    0,
		Board: "all",
	})
	defer Clients.Clear()
	cl.IP = "::1"

	stdThread := types.DatabaseThread{
		ID:       6,
		Subject:  "subject",
		Board:    "c",
		ImageCtr: 1,
		PostCtr:  1,
	}
	stdPost := types.DatabasePost{
		IP:  "::1",
		Log: [][]byte{},
		Post: types.Post{
			Editing: true,
			ID:      6,
			Name:    "name",
			Board:   "c",
			Image: &types.Image{
				Spoiler:     true,
				ImageCommon: stdJPEG,
				Name:        "foo",
			},
		},
	}

	req := threadCreationRequest{
		postCreationCommon: postCreationCommon{
			Name:     "name",
			Password: "123",
			Image: imageRequest{
				Name:    "foo.jpeg",
				Token:   token,
				Spoiler: true,
			},
		},
		Subject: "subject",
		Board:   "c",
	}
	data := marshalJSON(t, req)
	if err := insertThread(data, cl); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, `01{"code":0,"id":6}`)
	assertIP(t, 6, "::1")

	var (
		thread types.DatabaseThread
		post   types.DatabasePost
	)
	if err := db.One(db.FindThread(6), &thread); err != nil {
		t.Fatal(err)
	}
	if err := db.One(db.FindPost(6), &post); err != nil {
		t.Fatal(err)
	}

	// Pointers have to be dereferenced to be asserted
	assertDeepEquals(t, *post.Image, *stdPost.Image)

	// Normalize timestamps and pointer fields
	then := thread.BumpTime
	stdThread.BumpTime = then
	stdThread.ReplyTime = then
	stdPost.Time = then
	stdPost.LastUpdated = then
	stdPost.Password = post.Password
	stdPost.Image = post.Image

	assertDeepEquals(t, thread, stdThread)
	assertDeepEquals(t, post, stdPost)
	assertDeepEquals(t, cl.openPost, openPost{
		id:       6,
		op:       6,
		board:    "c",
		time:     then,
		hasImage: true,
	})
}

func testCreateThreadTextOnly(t *testing.T) {
	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	data := marshalJSON(t, sampleImagelessThreadCreationRequest)
	if err := insertThread(data, cl); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, `01{"code":0,"id":7}`)
	if cl.openPost.hasImage {
		t.Error("image inserted")
	}

	var noImage bool
	q := db.FindPost(7).HasFields("image").Not()
	if err := db.One(q, &noImage); err != nil {
		t.Fatal(err)
	}
	if !noImage {
		t.Error("image written to database")
	}
}

func populateMainTable(t *testing.T) {
	assertInsert(t, "main", []map[string]interface{}{
		{
			"id":      "info",
			"postCtr": 5,
		},
		{
			"id": "boardCtrs",
		},
	})
}

func writeBoardConfigs(t *testing.T, textOnly bool) {
	assertInsert(t, "boards", config.BoardConfigs{
		ID: "a",
		PostParseConfigs: config.PostParseConfigs{
			TextOnly: textOnly,
		},
	})
}

func assertIP(t *testing.T, id int64, ip string) {
	q := db.FindPost(id).Field("ip")
	var res string
	if err := db.One(q, &res); err != nil {
		t.Fatal(err)
	}
	if res != ip {
		t.Errorf("unexpcted ip: %s : %s", ip, res)
	}
}

func TestGetInvalidImage(t *testing.T) {
	assertTableClear(t, "images")

	const (
		name  = "foo.jpeg"
		token = "dasdasd-ad--dsad-ads-d-ad-"
	)

	cases := [...]struct {
		testName, token, name string
		err                   error
	}{
		{"empty token", "", name, errInvalidImageToken},
		{"token too long", genString(128), name, errInvalidImageToken},
		{"no image name", token, "", errNoImageName},
		{"image name too long", token, genString(201), errImageNameTooLong},
		{"no token in DB", token, name, errInvalidImageToken},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()
			if _, err := getImage(c.token, c.name, false); err != c.err {
				t.Fatalf("unexpected error %#v", err)
			}
		})
	}
}

func TestClosePreviousPostOnCreation(t *testing.T) {
	assertTableClear(t, "main", "threads", "posts", "boards")
	assertInsert(t, "posts", samplePost)
	(*config.Get()).Boards = []string{"a"}
	populateMainTable(t)
	writeBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}
	data := marshalJSON(t, sampleImagelessThreadCreationRequest)

	if err := insertThread(data, cl); err != nil {
		t.Fatal(err)
	}

	assertMessage(t, wcl, `01{"code":0,"id":6}`)
	assertRepLog(t, 2, append(strDummyLog, "062"))
	assertPostClosed(t, 2)
}

func TestPostCreationValidations(t *testing.T) {
	assertTableClear(t, "boards")
	writeBoardConfigs(t, false)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	Clients.Add(cl, SyncID{1, "a"})

	cases := [...]struct {
		testName, text, token, name string
	}{
		{"no token", "", "", "abc"},
		{"no image name", "", "abc", ""},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			req := replyCreationRequest{
				Body: c.text,
				postCreationCommon: postCreationCommon{
					Image: imageRequest{
						Name:  c.name,
						Token: c.token,
					},
				},
			}
			err := insertPost(marshalJSON(t, req), cl)
			if err != errNoTextOrImage {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	}
}

func TestPoctCreationOnLockedThread(t *testing.T) {
	assertTableClear(t, "boards", "threads")
	assertInsert(t, "threads", map[string]interface{}{
		"id":      1,
		"board":   "a",
		"postCtr": 0,
		"locked":  true,
	})
	writeBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	Clients.Add(cl, SyncID{1, "a"})
	defer Clients.Clear()

	req := replyCreationRequest{
		Body: "a",
	}
	if err := insertPost(marshalJSON(t, req), cl); err != errThreadIsLocked {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestPostCreation(t *testing.T) {
	prepareForPostCreation(t)
	writeBoardConfigs(t, false)
	assertInsert(t, "images", stdJPEG)
	_, token, err := db.NewImageToken(stdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	Clients.Add(cl, SyncID{1, "a"})
	defer Clients.Clear()
	cl.IP = "::1"

	req := replyCreationRequest{
		Body: "a",
		postCreationCommon: postCreationCommon{
			Password: "123",
			Email:    "wew lad",
			Image: imageRequest{
				Name:    "foo.jpeg",
				Token:   token,
				Spoiler: true,
			},
		},
	}

	if err := insertPost(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}

	assertMessage(t, wcl, "416")

	// Get the time value from the DB and normalize against it
	var then int64
	if err := db.One(db.FindPost(6).Field("time"), &then); err != nil {
		t.Fatal(err)
	}

	stdPost := types.Post{
		Editing:     true,
		ID:          6,
		OP:          1,
		Time:        then,
		LastUpdated: then,
		Body:        "a",
		Email:       "wew lad",
		Image: &types.Image{
			Name:        "foo",
			Spoiler:     true,
			ImageCommon: stdJPEG,
		},
	}

	var post types.Post
	if err := db.One(db.FindPost(6), &post); err != nil {
		t.Fatal(err)
	}
	assertDeepEquals(t, *post.Image, *stdPost.Image)
	stdPost.Image = post.Image
	assertDeepEquals(t, post, stdPost)

	assertIP(t, 6, "::1")

	// Assert thread was bumped
	type threadAttrs struct {
		PostCtr   int
		ImageCtr  int
		BumpTime  int64
		ReplyTime int64
	}

	var attrs threadAttrs
	q := db.FindThread(1).Pluck("postCtr", "imageCtr", "bumpTime", "replyTime")
	if err := db.One(q, &attrs); err != nil {
		t.Fatal(err)
	}
	stdAttrs := threadAttrs{
		PostCtr:   1,
		ImageCtr:  2,
		BumpTime:  then,
		ReplyTime: then,
	}
	if attrs != stdAttrs {
		logUnexpected(t, stdAttrs, attrs)
	}

	var boardCtr int
	q = db.GetMain("boardCtrs").Field("a")
	if err := db.One(q, &boardCtr); err != nil {
		t.Fatal(err)
	}
	if boardCtr != 1 {
		t.Errorf("unexpected board counter: %d", boardCtr)
	}

	assertDeepEquals(t, cl.openPost, openPost{
		id:         6,
		op:         1,
		time:       then,
		board:      "a",
		bodyLength: 1,
		Buffer:     *bytes.NewBuffer([]byte("a")),
		hasImage:   true,
	})
}

func prepareForPostCreation(t *testing.T) {
	now := time.Now().Unix()
	(*config.Get()).MaxBump = 500
	assertTableClear(t, "main", "boards", "threads", "posts")
	assertInsert(t, "threads", types.DatabaseThread{
		ID:        1,
		Board:     "a",
		PostCtr:   0,
		ImageCtr:  1,
		BumpTime:  now,
		ReplyTime: now,
	})
	populateMainTable(t)
}

func TestTextOnlyPostCreation(t *testing.T) {
	prepareForPostCreation(t)
	writeBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	Clients.Add(cl, SyncID{1, "a"})
	defer Clients.Clear()

	req := replyCreationRequest{
		Body: "a",
		postCreationCommon: postCreationCommon{
			Password: "123",
		},
	}

	if err := insertPost(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}

	// Assert image counter did not change
	assertImageCounter(t, 1, 1)

	// Assert no image in post
	var hasImage bool
	q := db.FindPost(6).HasFields("image")
	if err := db.One(q, &hasImage); err != nil {
		t.Fatal(err)
	}
	if hasImage {
		t.Error("DB post has image")
	}

	if cl.openPost.hasImage {
		t.Error("openPost has image")
	}
}

func assertImageCounter(t *testing.T, id int64, ctr int) {
	var res int
	q := db.FindThread(id).Field("imageCtr")
	if err := db.One(q, &res); err != nil {
		t.Fatal(err)
	}
	if res != ctr {
		t.Errorf("unexpected thrad image counter: %d : %d", ctr, res)
	}
}

func TestBumpLimit(t *testing.T) {
	assertTableClear(t, "main", "boards", "threads", "posts")

	(*config.Get()).MaxBump = 10
	then := time.Now().Add(-time.Minute).Unix()

	assertInsert(t, "threads", types.DatabaseThread{
		ID:        1,
		PostCtr:   10,
		Board:     "a",
		BumpTime:  then,
		ReplyTime: then,
	})
	populateMainTable(t)
	writeBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	Clients.Add(cl, SyncID{1, "a"})
	defer Clients.Clear()

	req := replyCreationRequest{
		Body: "a",
		postCreationCommon: postCreationCommon{
			Password: "123",
		},
	}
	if err := insertPost(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}

	var res types.DatabaseThread
	if err := db.One(db.FindThread(1), &res); err != nil {
		t.Fatal(err)
	}
	if res.BumpTime != then {
		t.Errorf("unexpected bump time: %d : %d", then, res.BumpTime)
	}
	if res.ReplyTime <= then {
		t.Error("invalid reply time")
	}
}

func TestSaging(t *testing.T) {
	assertTableClear(t, "main", "boards", "threads", "posts")

	(*config.Get()).MaxBump = 10
	then := time.Now().Add(-time.Minute).Unix()
	assertInsert(t, "threads", types.DatabaseThread{
		ID:        1,
		Board:     "a",
		BumpTime:  then,
		ReplyTime: then,
	})
	populateMainTable(t)
	writeBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	Clients.Add(cl, SyncID{1, "a"})
	defer Clients.Clear()

	req := replyCreationRequest{
		Body: "a",
		postCreationCommon: postCreationCommon{
			Password: "123",
			Email:    "sage",
		},
	}
	if err := insertPost(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}

	var res types.DatabaseThread
	q := db.FindThread(1).Pluck("replyTime", "bumpTime")
	if err := db.One(q, &res); err != nil {
		t.Fatal(err)
	}
	if res.BumpTime != then {
		t.Errorf("unexpected bump time: %d : %d", then, res.BumpTime)
	}
	if res.ReplyTime <= then {
		t.Error("invalid reply time")
	}
}

func TestPostCreationWithNewlines(t *testing.T) {
	assertTableClear(t, "main", "boards", "threads", "posts")

	(*config.Get()).MaxBump = 500
	assertInsert(t, "threads", types.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	populateMainTable(t)
	writeBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	Clients.Add(cl, SyncID{1, "a"})
	defer Clients.Clear()

	req := replyCreationRequest{
		Body: "abc\nd",
		postCreationCommon: postCreationCommon{
			Password: "123",
		},
	}
	if err := insertPost(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}

	var then int64
	if err := db.One(db.FindPost(6).Field("time"), &then); err != nil {
		t.Fatal(err)
	}

	log := []string{
		"03[6,10]",
		`05{"id":6,"start":0,"len":0,"text":"d"}`,
	}
	assertRepLog(t, 6, log)

	assertBody(t, 6, "abc\nd")
}
