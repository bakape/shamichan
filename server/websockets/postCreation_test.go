package websockets


// var (
// 	// JPEG sample image standard struct
// 	stdJPEG = types.ImageCommon{
// 		SHA1:     "012a2f912c9ee93ceb0ccb8684a29ec571990a94",
// 		FileType: 0,
// 		Dims:     [4]uint16{1, 1, 1, 1},
// 		MD5:      "60e41092581f7b329b057b8402caa8a7",
// 		Size:     300792,
// 	}

// 	sampleImagelessThreadCreationRequest = threadCreationRequest{
// 		postCreationCommon: postCreationCommon{
// 			Name:     "name",
// 			Password: "123",
// 		},
// 		Subject: "subject",
// 		Board:   "a",
// 	}
// )

// func (*DB) TestCreateThreadOnInvalidBoard(c *C) {
// 	req := threadCreationRequest{
// 		Board: "all",
// 	}
// 	err := insertThread(marshalJSON(req, c), new(Client))
// 	c.Assert(err, Equals, errInvalidBoard)
// }

// func (*DB) TestCreateThreadOnReadOnlyBoard(c *C) {
// 	q := r.Table("boards").Insert(config.BoardConfigs{
// 		ID: "a",
// 		PostParseConfigs: config.PostParseConfigs{
// 			ReadOnly: true,
// 		},
// 	})
// 	c.Assert(db.Write(q), IsNil)

// 	req := threadCreationRequest{
// 		Board: "a",
// 	}
// 	err := insertThread(marshalJSON(req, c), new(Client))
// 	c.Assert(err, Equals, errReadOnly)
// }

// func (*DB) TestThreadCreation(c *C) {
// 	populateMainTable(c)
// 	writeBoardConfigs(false, c)
// 	c.Assert(db.Write(r.Table("images").Insert(stdJPEG)), IsNil)
// 	_, token, err := db.NewImageToken(stdJPEG.SHA1)
// 	c.Assert(err, IsNil)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, wcl := sv.NewClient()
// 	Clients.Add(cl, SyncID{
// 		OP:    0,
// 		Board: "all",
// 	})
// 	cl.IP = "::1"

// 	std := types.DatabaseThread{
// 		ID:       6,
// 		Subject:  "subject",
// 		Board:    "a",
// 		ImageCtr: 1,
// 		Posts: map[int64]types.DatabasePost{
// 			6: types.DatabasePost{
// 				IP: "::1",
// 				Post: types.Post{
// 					Editing: true,
// 					ID:      6,
// 					Name:    "name",
// 					Image: &types.Image{
// 						Spoiler:     true,
// 						ImageCommon: stdJPEG,
// 						Name:        "foo",
// 					},
// 				},
// 			},
// 		},
// 		Log: [][]byte{},
// 	}

// 	req := threadCreationRequest{
// 		postCreationCommon: postCreationCommon{
// 			Name:     "name",
// 			Password: "123",
// 			Image: imageRequest{
// 				Name:    "foo.jpeg",
// 				Token:   token,
// 				Spoiler: true,
// 			},
// 		},
// 		Subject: "subject",
// 		Board:   "a",
// 	}
// 	data := marshalJSON(req, c)
// 	c.Assert(insertThread(data, cl), IsNil)
// 	assertMessage(wcl, []byte(`01{"code":0,"id":6}`), c)
// 	assertIP(6, "::1", c)

// 	var thread types.DatabaseThread
// 	c.Assert(db.One(r.Table("threads").Get(6), &thread), IsNil)

// 	// Pointers have to be dereferenced to be asserted
// 	c.Assert(*thread.Posts[6].Image, DeepEquals, *std.Posts[6].Image)

// 	// Normalize timestamps and pointer fields
// 	then := thread.BumpTime
// 	std.BumpTime = then
// 	std.ReplyTime = then
// 	post := std.Posts[6]
// 	post.Time = then
// 	post.Password = thread.Posts[6].Password
// 	post.Image = thread.Posts[6].Image
// 	std.Posts[6] = post

// 	c.Assert(thread, DeepEquals, std)

// 	c.Assert(cl.openPost, DeepEquals, openPost{
// 		id:       6,
// 		op:       6,
// 		board:    "a",
// 		time:     then,
// 		hasImage: true,
// 	})
// }

// func populateMainTable(c *C) {
// 	mains := []map[string]interface{}{
// 		{
// 			"id":      "info",
// 			"postCtr": 5,
// 		},
// 		{
// 			"id": "boardCtrs",
// 		},
// 	}
// 	c.Assert(db.Write(r.Table("main").Insert(mains)), IsNil)
// }

// func writeBoardConfigs(textOnly bool, c *C) {
// 	conf := config.BoardConfigs{
// 		ID: "a",
// 		PostParseConfigs: config.PostParseConfigs{
// 			TextOnly: textOnly,
// 		},
// 	}
// 	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)
// }

// func assertIP(id int64, ip string, c *C) {
// 	q := db.FindPost(id).Field("ip")
// 	var res string
// 	c.Assert(db.One(q, &res), IsNil)
// 	c.Assert(res, Equals, ip)
// }

// func (*DB) TestTextOnlyThreadCreation(c *C) {
// 	populateMainTable(c)
// 	writeBoardConfigs(true, c)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, wcl := sv.NewClient()
// 	data := marshalJSON(sampleImagelessThreadCreationRequest, c)
// 	c.Assert(insertThread(data, cl), IsNil)
// 	assertMessage(wcl, []byte(`01{"code":0,"id":6}`), c)
// 	c.Assert(cl.openPost.hasImage, Equals, false)

// 	var post types.Post
// 	c.Assert(db.One(db.FindPost(6), &post), IsNil)
// 	c.Assert(post.Image, IsNil)
// }

// func (*DB) TestGetInvalidImage(c *C) {
// 	const (
// 		name  = "foo.jpeg"
// 		token = "dasdasd-ad--dsad-ads-d-ad-"
// 	)
// 	r128, err := auth.RandomID(128)
// 	c.Assert(err, IsNil)
// 	r128 = r128[:128]
// 	r201, err := auth.RandomID(201)
// 	c.Assert(err, IsNil)
// 	r201 = r201[:201]

// 	samples := [...]struct {
// 		token, name string
// 		err         error
// 	}{
// 		{"", name, errInvalidImageToken},
// 		{r128, name, errInvalidImageToken},
// 		{token, "", errNoImageName},
// 		{token, r201, errImageNameTooLong},
// 		{token, name, errInvalidImageToken}, // No token in the database
// 	}

// 	for _, s := range samples {
// 		_, err := getImage(s.token, s.name, false)
// 		c.Assert(err, Equals, s.err)
// 	}
// }

// func (*DB) TestClosePreviousPostOnCreation(c *C) {
// 	thread := sampleThread
// 	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)
// 	populateMainTable(c)
// 	writeBoardConfigs(true, c)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, wcl := sv.NewClient()
// 	cl.openPost = openPost{
// 		id:         2,
// 		op:         1,
// 		bodyLength: 3,
// 		board:      "a",
// 		time:       time.Now().Unix(),
// 		Buffer:     *bytes.NewBuffer([]byte("abc")),
// 	}
// 	data := marshalJSON(sampleImagelessThreadCreationRequest, c)

// 	c.Assert(insertThread(data, cl), IsNil)

// 	assertMessage(wcl, []byte(`01{"code":0,"id":6}`), c)
// 	assertRepLog(2, append(strDummyLog, "062"), c)
// 	assertPostClosed(2, c)
// }

// func (*DB) TestPostCreationValidations(c *C) {
// 	writeBoardConfigs(false, c)
// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, _ := sv.NewClient()
// 	Clients.Add(cl, SyncID{1, "a"})

// 	samples := [...]struct {
// 		text, token, name string
// 	}{
// 		{"", "", "abc"},
// 		{"", "abc", ""},
// 	}

// 	for _, s := range samples {
// 		req := replyCreationRequest{
// 			Body: s.text,
// 			postCreationCommon: postCreationCommon{
// 				Image: imageRequest{
// 					Name:  s.name,
// 					Token: s.token,
// 				},
// 			},
// 		}
// 		err := insertPost(marshalJSON(req, c), cl)
// 		c.Assert(err, Equals, errNoTextOrImage)
// 	}
// }

// func (*DB) TestPoctCreationOnLockedThread(c *C) {
// 	thread := map[string]interface{}{
// 		"id":      1,
// 		"postCtr": 0,
// 		"locked":  true,
// 	}
// 	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)
// 	writeBoardConfigs(true, c)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, _ := sv.NewClient()
// 	Clients.Add(cl, SyncID{1, "a"})

// 	req := replyCreationRequest{
// 		Body: "a",
// 	}
// 	data := marshalJSON(req, c)
// 	c.Assert(insertPost(data, cl), Equals, errThreadIsLocked)
// }

// func (*DB) TestPostCreation(c *C) {
// 	now := prepareForPostCreation(c)
// 	writeBoardConfigs(false, c)
// 	c.Assert(db.Write(r.Table("images").Insert(stdJPEG)), IsNil)
// 	_, token, err := db.NewImageToken(stdJPEG.SHA1)
// 	c.Assert(err, IsNil)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, wcl := sv.NewClient()
// 	Clients.Add(cl, SyncID{1, "a"})
// 	cl.IP = "::1"

// 	req := replyCreationRequest{
// 		Body: "a",
// 		postCreationCommon: postCreationCommon{
// 			Password: "123",
// 			Email:    "wew lad",
// 			Image: imageRequest{
// 				Name:    "foo.jpeg",
// 				Token:   token,
// 				Spoiler: true,
// 			},
// 		},
// 	}
// 	data := marshalJSON(req, c)

// 	c.Assert(insertPost(data, cl), IsNil)

// 	assertMessage(wcl, []byte("416"), c)

// 	// Get the time value from the DB and normalize against it
// 	var then int64
// 	c.Assert(db.One(db.FindPost(6).Field("time"), &then), IsNil)
// 	c.Assert(then >= now, Equals, true)

// 	post := types.Post{
// 		Editing: true,
// 		ID:      6,
// 		Time:    then,
// 		Body:    "a",
// 		Email:   "wew lad",
// 		Image: &types.Image{
// 			Name:        "foo",
// 			Spoiler:     true,
// 			ImageCommon: stdJPEG,
// 		},
// 	}
// 	stdMsg, err := EncodeMessage(MessageInsertPost, post)
// 	c.Assert(err, IsNil)

// 	assertRepLog(6, append(strDummyLog, string(stdMsg)), c)
// 	assertIP(6, "::1", c)

// 	// Assert thread was bumped
// 	type threadAttrs struct {
// 		PostCtr   int   `gorethink:"postCtr"`
// 		ImageCtr  int   `gorethink:"imageCtr"`
// 		BumpTime  int64 `gorethink:"bumpTime"`
// 		ReplyTime int64 `gorethink:"replyTime"`
// 	}

// 	var attrs threadAttrs
// 	q := db.FindParentThread(6).
// 		Pluck("postCtr", "imageCtr", "bumpTime", "replyTime")
// 	c.Assert(db.One(q, &attrs), IsNil)
// 	c.Assert(attrs, DeepEquals, threadAttrs{
// 		PostCtr:   1,
// 		ImageCtr:  2,
// 		BumpTime:  then,
// 		ReplyTime: then,
// 	})

// 	var boardCtr int
// 	q = db.GetMain("boardCtrs").Field("a")
// 	c.Assert(db.One(q, &boardCtr), IsNil)
// 	c.Assert(boardCtr, Equals, 1)

// 	c.Assert(cl.openPost, DeepEquals, openPost{
// 		id:         6,
// 		op:         1,
// 		time:       then,
// 		board:      "a",
// 		bodyLength: 1,
// 		Buffer:     *bytes.NewBuffer([]byte("a")),
// 		hasImage:   true,
// 	})
// }

// func prepareForPostCreation(c *C) int64 {
// 	now := time.Now().Unix()
// 	(*config.Get()).MaxBump = 500
// 	thread := types.DatabaseThread{
// 		ID:        1,
// 		Board:     "a",
// 		PostCtr:   0,
// 		ImageCtr:  1,
// 		Log:       dummyLog,
// 		BumpTime:  now,
// 		ReplyTime: now,
// 		Posts: map[int64]types.DatabasePost{
// 			1: {
// 				Post: types.Post{
// 					Time: now,
// 					ID:   1,
// 				},
// 			},
// 		},
// 	}
// 	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)
// 	populateMainTable(c)
// 	return now
// }

// func (*DB) TestTextOnlyPostCreation(c *C) {
// 	prepareForPostCreation(c)
// 	writeBoardConfigs(true, c)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, _ := sv.NewClient()
// 	Clients.Add(cl, SyncID{1, "a"})

// 	req := replyCreationRequest{
// 		Body: "a",
// 		postCreationCommon: postCreationCommon{
// 			Password: "123",
// 		},
// 	}
// 	data := marshalJSON(req, c)

// 	c.Assert(insertPost(data, cl), IsNil)

// 	// Assert image counter did not change
// 	assertImageCounter(6, 1, c)

// 	// Assert no image in post
// 	var hasImage bool
// 	q := db.FindPost(6).HasFields("image")
// 	c.Assert(db.One(q, &hasImage), IsNil)
// 	c.Assert(hasImage, Equals, false)

// 	c.Assert(cl.openPost.hasImage, Equals, false)
// }

// func assertImageCounter(id int64, ctr int, c *C) {
// 	var res int
// 	q := db.FindParentThread(id).Field("imageCtr")
// 	c.Assert(db.One(q, &res), IsNil)
// 	c.Assert(res, Equals, ctr)
// }

// func (*DB) TestBumpLimit(c *C) {
// 	(*config.Get()).MaxBump = 10
// 	then := time.Now().Add(-time.Minute).Unix()
// 	thread := types.DatabaseThread{
// 		ID:        1,
// 		PostCtr:   10,
// 		Board:     "a",
// 		BumpTime:  then,
// 		ReplyTime: then,
// 	}
// 	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)
// 	populateMainTable(c)
// 	writeBoardConfigs(true, c)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, _ := sv.NewClient()
// 	Clients.Add(cl, SyncID{1, "a"})

// 	req := replyCreationRequest{
// 		Body: "a",
// 		postCreationCommon: postCreationCommon{
// 			Password: "123",
// 		},
// 	}
// 	data := marshalJSON(req, c)
// 	c.Assert(insertPost(data, cl), IsNil)

// 	var res types.DatabaseThread
// 	c.Assert(db.One(db.FindParentThread(6), &res), IsNil)
// 	c.Assert(res.BumpTime, Equals, then)
// 	c.Assert(res.ReplyTime > then, Equals, true)
// }

// func (*DB) TestSaging(c *C) {
// 	(*config.Get()).MaxBump = 10
// 	then := time.Now().Add(-time.Minute).Unix()
// 	thread := types.DatabaseThread{
// 		ID:        1,
// 		Board:     "a",
// 		BumpTime:  then,
// 		ReplyTime: then,
// 	}
// 	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)
// 	populateMainTable(c)
// 	writeBoardConfigs(true, c)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, _ := sv.NewClient()
// 	Clients.Add(cl, SyncID{1, "a"})

// 	req := replyCreationRequest{
// 		Body: "a",
// 		postCreationCommon: postCreationCommon{
// 			Password: "123",
// 			Email:    "sage",
// 		},
// 	}
// 	data := marshalJSON(req, c)
// 	c.Assert(insertPost(data, cl), IsNil)

// 	var res types.DatabaseThread
// 	q := db.FindParentThread(6).Pluck("replyTime", "bumpTime")
// 	c.Assert(db.One(q, &res), IsNil)
// 	c.Assert(res.BumpTime, Equals, then)
// 	c.Assert(res.ReplyTime > then, Equals, true)
// }

// func (*DB) TestPostCreationWithNewlines(c *C) {
// 	(*config.Get()).MaxBump = 500
// 	thread := types.DatabaseThread{
// 		ID:    1,
// 		Board: "a",
// 	}
// 	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)
// 	populateMainTable(c)
// 	writeBoardConfigs(true, c)

// 	sv := newWSServer(c)
// 	defer sv.Close()
// 	cl, _ := sv.NewClient()
// 	Clients.Add(cl, SyncID{1, "a"})

// 	req := replyCreationRequest{
// 		Body: "abc\nd",
// 		postCreationCommon: postCreationCommon{
// 			Password: "123",
// 		},
// 	}
// 	data := marshalJSON(req, c)
// 	c.Assert(insertPost(data, cl), IsNil)

// 	var then int64
// 	c.Assert(db.One(db.FindPost(6).Field("time"), &then), IsNil)

// 	post := types.Post{
// 		Editing: true,
// 		ID:      6,
// 		Time:    then,
// 		Body:    "abc",
// 	}
// 	postMsg, err := EncodeMessage(MessageInsertPost, post)
// 	c.Assert(err, IsNil)
// 	log := []string{
// 		string(postMsg),
// 		"03[6,10]",
// 		`05{"id":6,"start":0,"len":0,"text":"d"}`,
// 	}
// 	assertRepLog(6, log, c)

// 	assertBody(6, "abc\nd", c)
// }
