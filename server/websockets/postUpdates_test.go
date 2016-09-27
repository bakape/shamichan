package websockets

import (
	"bytes"
	"strings"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

// Sample wall of text
const longPost = `Shut the fuck up. I'm so tired of being disrespected on this
goddamn website. All I wanted to do was post my opinion. MY OPINION. But no,
you little bastards think it's "hilarious" to mock those with good opinions.
My opinion. while not absolute, is definitely worth the respect to formulate
an ACTUAL FUCKING RESPONSE AND NOT JUST A SHORT MEME OF A REPLY. I've been on
this site for 6 months: 6 MONTHS and I have never felt this wronged. It boils
me up that I could spend so much time thinking and putting effort into things
while you shits sit around (probably jerking off to Gardevoir or whatever
furbait you like) and make fun of the intellectuals of this world. You're
laughing at me? Good for fucking you. Literally no one cares that your little
brain is to underdeveloped and rotted to comprehend this game...THIS GREAT
GREAT GAME. I could sit here all day whining, but I won't. I'm NOT a whiner.
I'm a realist and an intellectual. I know when to call it quits and to leave
the babybrains to themselves. I'm done with this goddamn site and you goddamn
immature children. I have lived my life up until this point having to deal
with memesters and idiots like you. I know how you work. I know that you all
think you're "epik trolls" but you're not. You think you baited me? NAH. I've
never taken any bait. This is my 100% real opinion divorced from anger. I'm
calm, I'm serene. I LAUGH when people imply I'm intellectually low enough to
take bait. I always choose to reply just to spite you. I won. I've always won.
Losing is not in my skillset. So you're probably gonna reply "lol epik
trolled" or "u mad bro" but once you've done that you've shown me I've won.
I've tricked the trickster and conquered memery. I live everyday growing
stronger to fight you plebs and low level trolls who are probably 11 (baby,
you gotta be 18 to use 4chan). But whatever, I digress. It's just fucking
annoying that I'm never taken serious on this site, goddamn.`

var (
	dummyLog = [][]byte{
		{102, 111, 111},
		{98, 97, 114},
	}

	strDummyLog = []string{
		"foo",
		"bar",
	}

	sampleThread = types.DatabaseThread{
		ID:  1,
		Log: dummyLog,
		Posts: map[int64]types.DatabasePost{
			2: {
				Post: types.Post{
					Editing: true,
					ID:      2,
					Body:    "abc",
				},
			},
		},
	}
)

func (*DB) TestWriteBacklinks(c *C) {
	threads := []types.DatabaseThread{
		{
			ID: 1,
			Posts: map[int64]types.DatabasePost{
				1: {
					Post: types.Post{
						ID: 1,
					},
				},
				2: {
					Post: types.Post{
						ID: 2,
					},
				},
			},
			Log: dummyLog,
		},
		{
			ID: 5,
			Posts: map[int64]types.DatabasePost{
				7: {
					Post: types.Post{
						ID: 7,
					},
				},
			},
			Log: dummyLog,
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(threads)), IsNil)

	for _, dest := range [...]int64{1, 2, 7, 8} {
		c.Assert(writeBacklink(10, 9, "a", dest), IsNil)
	}

	// Assert each existong post had a backlink inserted
	std := types.Link{
		OP:    9,
		Board: "a",
	}

	for _, id := range [...]int64{1, 2, 7} {
		var link types.Link
		q := db.FindPost(id).Field("backlinks").Field("10")
		c.Assert(db.One(q, &link), IsNil)
		c.Assert(link, Equals, std)

		stdMsg, err := EncodeMessage(MessageBacklink, linkMessage{
			ID: id,
			Links: types.LinkMap{
				10: {
					OP:    9,
					Board: "a",
				},
			},
		})
		c.Assert(err, IsNil)

		var constains bool
		q = db.FindParentThread(id).Field("log").Contains(stdMsg)
		c.Assert(db.One(q, &constains), IsNil)
		c.Assert(constains, Equals, true)
	}
}

func (*DB) TestNoOpenPost(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	fns := [...]func([]byte, *Client) error{
		appendRune, backspace, closePost, spliceText, insertImage,
	}
	for _, fn := range fns {
		cl, _ := sv.NewClient()
		c.Assert(fn(nil, cl), Equals, errNoPostOpen)
	}
}

func (*DB) TestLineEmpty(c *C) {
	fns := [...]func([]byte, *Client) error{backspace}
	sv := newWSServer(c)
	defer sv.Close()

	for _, fn := range fns {
		cl, _ := sv.NewClient()
		cl.openPost.id = 1
		cl.openPost.time = time.Now().Unix()
		c.Assert(fn(nil, cl), Equals, errLineEmpty)
	}
}

func (*DB) TestAppendBodyTooLong(c *C) {
	sv := newWSServer(c)
	defer sv.Close()

	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         1,
		time:       time.Now().Unix(),
		bodyLength: parser.MaxLengthBody,
	}
	c.Assert(appendRune(nil, cl), Equals, parser.ErrBodyTooLong)
}

func (*DB) TestAppendRune(c *C) {
	c.Assert(db.Write(r.Table("threads").Insert(sampleThread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	c.Assert(appendRune([]byte("100"), cl), IsNil)

	c.Assert(cl.openPost.bodyLength, Equals, 4)
	c.Assert(cl.openPost.String(), Equals, "abcd")
	assertBody(2, "abcd", c)

	assertRepLog(2, append(strDummyLog, `03[2,100]`), c)
}

func assertBody(id int64, body string, c *C) {
	var res string
	q := db.FindPost(id).Field("body")
	c.Assert(db.One(q, &res), IsNil)
	c.Assert(res, Equals, body)
}

func assertRepLog(id int64, log []string, c *C) {
	var res [][]byte
	q := db.FindParentThread(id).Field("log")
	c.Assert(db.All(q, &res), IsNil)

	strRes := make([]string, len(res))
	for i := range res {
		strRes[i] = string(res[i])
	}
	c.Assert(strRes, DeepEquals, log)
}

func (*DB) TestAppendNewline(c *C) {
	c.Assert(db.Write(r.Table("threads").Insert(sampleThread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}
	writeBoardConfigs(false, c)

	c.Assert(appendRune([]byte("10"), cl), IsNil)

	c.Assert(cl.openPost.bodyLength, Equals, 4)
	c.Assert(cl.openPost.String(), Equals, "")
	assertBody(2, "abc\n", c)
	assertRepLog(2, append(strDummyLog, "03[2,10]"), c)
}

func (*DB) TestAppendNewlineWithHashCommand(c *C) {
	thread := types.DatabaseThread{
		ID:  1,
		Log: dummyLog,
		Posts: map[int64]types.DatabasePost{
			2: {
				Post: types.Post{
					ID:   2,
					Body: "#flip",
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("#flip")),
	}

	conf := config.BoardConfigs{
		ID: "a",
		PostParseConfigs: config.PostParseConfigs{
			HashCommands: true,
		},
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	c.Assert(appendRune([]byte("10"), cl), IsNil)

	var typ int
	q := db.FindPost(2).Field("commands").AtIndex(0).Field("type")
	c.Assert(db.One(q, &typ), IsNil)
	c.Assert(typ, Equals, int(types.Flip))

	var log []byte
	q = db.FindParentThread(2).Field("log").Nth(-1)
	c.Assert(db.One(q, &log), IsNil)
	c.Assert(string(log), Equals, "03[2,10]")

	q = db.FindParentThread(2).Field("log").Nth(-2)
	c.Assert(db.One(q, &log), IsNil)
	c.Assert(string(log), Matches, `09{"id":2,"type":1,"val":(?:true|false)}`)
}

func (*DB) TestAppendNewlineWithLinks(c *C) {
	threads := []types.DatabaseThread{
		{
			ID:    1,
			Board: "a",
			Log:   [][]byte{},
			Posts: map[int64]types.DatabasePost{
				2: {
					Post: types.Post{
						ID:   2,
						Body: " >>22 ",
					},
				},
			},
		},
		{
			ID:    21,
			Board: "c",
			Log:   [][]byte{},
			Posts: map[int64]types.DatabasePost{
				22: {
					Post: types.Post{
						ID: 22,
					},
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(threads)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte(" >>22 ")),
	}
	writeBoardConfigs(false, c)

	c.Assert(appendRune([]byte("10"), cl), IsNil)

	std := [...]struct {
		id    int64
		log   []string
		field string
		val   types.LinkMap
	}{
		{
			id: 2,
			log: []string{
				`07{"id":2,"links":{"22":{"op":21,"board":"c"}}}`,
				`03[2,10]`,
			},
			field: "links",
			val: types.LinkMap{
				22: {
					OP:    21,
					Board: "c",
				},
			},
		},
		{
			id: 22,
			log: []string{
				`08{"id":22,"links":{"2":{"op":1,"board":"a"}}}`,
			},
			field: "backlinks",
			val: types.LinkMap{
				2: {
					OP:    1,
					Board: "a",
				},
			},
		},
	}
	for _, s := range std {
		assertRepLog(s.id, s.log, c)

		var links types.LinkMap
		q := db.FindPost(s.id).Field(s.field)
		c.Assert(db.One(q, &links), IsNil)
		c.Assert(links, DeepEquals, s.val)
	}
}

func (*DB) TestBackspace(c *C) {
	c.Assert(db.Write(r.Table("threads").Insert(sampleThread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	c.Assert(backspace([]byte{}, cl), IsNil)

	c.Assert(cl.openPost.String(), Equals, "ab")
	c.Assert(cl.openPost.bodyLength, Equals, 2)

	assertRepLog(2, append(strDummyLog, "042"), c)
	assertBody(2, "ab", c)
}

func (*DB) TestClosePost(c *C) {
	c.Assert(db.Write(r.Table("threads").Insert(sampleThread)), IsNil)
	writeBoardConfigs(false, c)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	c.Assert(closePost([]byte{}, cl), IsNil)

	c.Assert(cl.openPost, DeepEquals, openPost{})
	assertRepLog(2, append(strDummyLog, "062"), c)
	assertBody(2, "abc", c)
	assertPostClosed(2, c)
}

func assertPostClosed(id int64, c *C) {
	var editing bool
	q := db.FindPost(id).Field("editing")
	c.Assert(db.One(q, &editing), IsNil)
	c.Assert(editing, Equals, false)
}

func (*DB) TestSpliceValidityChecks(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:   2,
		time: time.Now().Unix(),
	}

	var tooLong string
	for i := 0; i < 2001; i++ {
		tooLong += "a"
	}

	samples := [...]struct {
		start, len int
		text, line string
		err        error
	}{
		{-1, 1, "", "", errInvalidSpliceCoords},
		{2, 1, "", "abc", errInvalidSpliceCoords},
		{0, 0, "", "", errSpliceNOOP},
		{0, 0, tooLong, "", errSpliceTooLong},
	}
	for _, s := range samples {
		req := spliceRequest{
			Start: s.start,
			Len:   s.len,
			Text:  s.text,
		}
		c.Assert(spliceText(marshalJSON(req, c), cl), Equals, s.err)
	}
}

func (*DB) TestSplice(c *C) {
	const longSplice = `Never gonna give you up ` +
		`Never gonna let you down ` +
		`Never gonna run around and desert you ` +
		`Never gonna make you cry ` +
		`Never gonna say goodbye ` +
		`Never gonna tell a lie and hurt you `

	sv := newWSServer(c)
	defer sv.Close()
	writeBoardConfigs(false, c)

	samples := [...]struct {
		start, len        int
		text, init, final string
		log               []string
	}{
		{
			start: 0,
			len:   0,
			text:  "abc",
			init:  "",
			final: "abc",
			log:   []string{`05{"id":2,"start":0,"len":0,"text":"abc"}`},
		},
		{
			start: 0,
			len:   1,
			text:  "",
			init:  "abc",
			final: "bc",
			log:   []string{`05{"id":2,"start":0,"len":1,"text":""}`},
		},
		{
			start: 2,
			len:   -1,
			text:  "abc",
			init:  "abcd",
			final: "ababc",
			log:   []string{`05{"id":2,"start":2,"len":-1,"text":"abc"}`},
		},
		{
			start: 2,
			len:   -1,
			text:  "abc",
			init:  "ab",
			final: "ababc",
			log:   []string{`05{"id":2,"start":2,"len":-1,"text":"abc"}`},
		},
		{
			start: 2,
			len:   3,
			text:  "abcdefg",
			init:  "00\n012345",
			final: "00\n01abcdefg5",
			log:   []string{`05{"id":2,"start":2,"len":3,"text":"abcdefg"}`},
		},
		{
			start: 52,
			len:   0,
			text:  longSplice,
			init:  longPost,
			final: longPost[:1943] + longSplice[:57],
			log: []string{
				`05{"id":2,"start":52,"len":-1,"text":"Never gonna give you` +
					` up Never gonna let you down Never go"}`,
			},
		},
		{
			start: 60,
			len:   0,
			text:  longSplice + "\n",
			init:  longPost,
			final: longPost + longSplice[:49],
			log: []string{
				`05{"id":2,"start":60,"len":-1,"text":"Never gonna give you` +
					` up Never gonna let you down "}`,
			},
		},
		{
			start: 2,
			len:   1,
			text:  "abc\nefg",
			init:  "00\n012345",
			final: "00\n01abc\nefg345",
			log: []string{
				`05{"id":2,"start":2,"len":-1,"text":"abc"}`,
				"03[2,10]",
				`05{"id":2,"start":0,"len":0,"text":"efg345"}`,
			},
		},
		{
			start: 2,
			len:   0,
			text:  "\n",
			init:  "012345",
			final: "01\n2345",
			log: []string{
				`05{"id":2,"start":2,"len":-1,"text":""}`,
				"03[2,10]",
				`05{"id":2,"start":0,"len":0,"text":"2345"}`,
			},
		},
	}

	for _, s := range samples {
		thread := types.DatabaseThread{
			ID:  1,
			Log: [][]byte{},
			Posts: map[int64]types.DatabasePost{
				2: {
					Post: types.Post{
						Editing: true,
						ID:      2,
						Body:    s.init,
					},
				},
			},
		}
		c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

		cl, _ := sv.NewClient()
		cl.openPost = openPost{
			id:         2,
			op:         1,
			bodyLength: len(s.init),
			board:      "a",
			time:       time.Now().Unix(),
			Buffer:     *bytes.NewBuffer([]byte(lastLine(s.init))),
		}

		req := spliceRequest{
			Start: s.start,
			Len:   s.len,
			Text:  s.text,
		}
		data := marshalJSON(req, c)
		c.Assert(spliceText(data, cl), IsNil)

		c.Assert(cl.openPost.String(), Equals, lastLine(s.final))
		c.Assert(cl.openPost.bodyLength, Equals, len(s.final))
		assertBody(2, s.final, c)
		assertRepLog(2, s.log, c)

		// Clean up
		c.Assert(db.Write(r.Table("threads").Delete()), IsNil)
	}
}

func lastLine(s string) string {
	lines := strings.Split(s, "\n")
	return lines[len(lines)-1]
}

func (*DB) TestCloseOldOpenPost(c *C) {
	then := time.Now().Add(time.Minute * -30).Unix()
	thread := types.DatabaseThread{
		ID:  1,
		Log: [][]byte{},
		Posts: map[int64]types.DatabasePost{
			1: {
				Post: types.Post{
					Editing: true,
					ID:      1,
					Time:    then,
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:   1,
		op:   1,
		time: then,
	}

	has, err := cl.hasPost()
	c.Assert(has, Equals, false)
	c.Assert(err, IsNil)

	var editing bool
	c.Assert(db.One(db.FindPost(1).Field("editing"), &editing), IsNil)
	c.Assert(editing, Equals, false)

	assertRepLog(1, []string{"061"}, c)
}

func (*DB) TestInsertImageIntoPostWithImage(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:       1,
		time:     time.Now().Unix(),
		hasImage: true,
	}
	c.Assert(insertImage(nil, cl), Equals, errHasImage)
}

func (*DB) TestInsertImageOnTextOnlyBoard(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:    1,
		board: "a",
		time:  time.Now().Unix(),
	}
	writeBoardConfigs(true, c)

	req := imageRequest{
		Name:  "foo.jpeg",
		Token: "123",
	}
	data := marshalJSON(req, c)
	c.Assert(insertImage(data, cl), Equals, errTextOnly)
}

func (*DB) TestInsertImage(c *C) {
	writeBoardConfigs(false, c)
	thread := types.DatabaseThread{
		ID:      1,
		Board:   "a",
		PostCtr: 1,
		Posts: map[int64]types.DatabasePost{
			2: {
				Post: types.Post{
					ID: 2,
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:    2,
		board: "a",
		op:    1,
		time:  time.Now().Unix(),
	}

	c.Assert(db.Write(r.Table("images").Insert(stdJPEG)), IsNil)
	_, token, err := db.NewImageToken(stdJPEG.SHA1)
	c.Assert(err, IsNil)

	req := imageRequest{
		Name:  "foo.jpeg",
		Token: token,
	}
	data := marshalJSON(req, c)
	c.Assert(insertImage(data, cl), IsNil)

	std := types.Image{
		Name:        "foo",
		ImageCommon: stdJPEG,
	}
	msg, err := EncodeMessage(MessageInsertImage, imageMessage{
		ID:    2,
		Image: std,
	})
	c.Assert(err, IsNil)
	assertRepLog(2, []string{string(msg)}, c)
	assertImageCounter(2, 1, c)

	var res types.Image
	q := db.FindPost(2).Field("image")
	c.Assert(db.One(q, &res), IsNil)
	c.Assert(res, Equals, std)

	c.Assert(cl.openPost.hasImage, Equals, true)
}
