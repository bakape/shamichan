package websockets

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"testing"
	"time"

	"unicode/utf8"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
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
	dummyLog = []string{
		"foo",
		"bar",
	}

	samplePost = common.DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Editing: true,
				ID:      2,
				Body:    "abc",
			},
			OP:    1,
			Board: "a",
		},
		Log: dummyLog,
	}
)

func TestWriteBacklinks(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 1,
				},
			},
			Log: dummyLog,
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 2,
				},
			},
			Log: dummyLog,
		},
	})

	for _, dest := range [...]uint64{1, 2, 8} {
		if err := writeBacklink(10, 9, "a", dest); err != nil {
			t.Fatalf("write post %d backlink: %s", dest, err)
		}
	}

	// Assert each existing post had a backlink inserted
	std := common.Link{
		OP:    9,
		Board: "a",
	}

	for _, i := range [...]uint64{1, 2} {
		id := i
		t.Run(fmt.Sprintf("post %d", id), func(t *testing.T) {
			t.Parallel()

			var link common.Link
			q := db.FindPost(id).Field("backlinks").Field("10")
			if err := db.One(q, &link); err != nil {
				t.Fatal(err)
			}
			if link != std {
				LogUnexpected(t, std, link)
			}

			msg, err := EncodeMessage(MessageBacklink, linkMessage{
				ID: id,
				Links: common.LinkMap{
					10: {
						OP:    9,
						Board: "a",
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			var written bool
			q = db.FindPost(id).Field("log").Eq(append(dummyLog, msg))
			if err := db.One(q, &written); err != nil {
				t.Fatal(err)
			}
			if !written {
				t.Error("no message in replication log")
			}
		})
	}
}

func TestNoOpenPost(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()

	fns := [...]func([]byte, *Client) error{
		appendRune, backspace, closePost, spliceText, insertImage,
	}
	for _, fn := range fns {
		cl, _ := sv.NewClient()
		if err := fn(nil, cl); err != errNoPostOpen {
			t.Errorf("unexpected error by %s: %s", funcName(fn), err)
		}
	}
}

func funcName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func TestLineEmpty(t *testing.T) {
	t.Parallel()

	fns := [...]func([]byte, *Client) error{backspace}

	sv := newWSServer(t)
	defer sv.Close()

	for _, fn := range fns {
		cl, _ := sv.NewClient()
		cl.openPost.id = 1
		cl.openPost.time = time.Now().Unix()
		if err := fn(nil, cl); err != errLineEmpty {
			t.Errorf("unexpected error by %s: %s", funcName(fn), err)
		}
	}
}

func TestAppendBodyTooLong(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()

	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         1,
		time:       time.Now().Unix(),
		bodyLength: common.MaxLenBody,
	}
	if err := appendRune(nil, cl); err != common.ErrBodyTooLong {
		UnexpectedError(t, err)
	}
}

func TestAppendRune(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", samplePost)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	if err := appendRune([]byte("100"), cl); err != nil {
		t.Fatal(err)
	}

	assertOpenPost(t, cl, 4, "abcd")
	assertBody(t, 2, "abcd")
	assertRepLog(t, 2, append(dummyLog, `03[2,100]`))
}

func assertOpenPost(t *testing.T, cl *Client, len int, buf string) {
	if l := cl.openPost.bodyLength; l != len {
		t.Errorf("unexpected openPost body length: %d", l)
	}
	if s := cl.openPost.String(); s != buf {
		t.Errorf("unexpected openPost buffer contents: `%s`", s)
	}
}

func assertBody(t *testing.T, id uint64, body string) {
	var res string
	q := db.FindPost(id).Field("body")
	if err := db.One(q, &res); err != nil {
		t.Fatal(err)
	}
	if res != body {
		LogUnexpected(t, body, res)
	}
}

func assertRepLog(t *testing.T, id uint64, log []string) {
	var res []string
	q := db.FindPost(id).Field("log")
	if err := db.All(q, &res); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, res, log)
}

func BenchmarkAppend(b *testing.B) {
	assertTableClear(b, "posts")
	assertInsert(b, "posts", samplePost)

	sv := newWSServer(b)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := appendRune([]byte("100"), cl); err != nil {
			b.Fatal(err)
		}
	}
}

func TestAppendNewline(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", samplePost)

	sv := newWSServer(t)
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
	setBoardConfigs(t, false)

	if err := appendRune([]byte("10"), cl); err != nil {
		t.Fatal(err)
	}

	assertOpenPost(t, cl, 4, "")
	assertBody(t, 2, "abc\n")
	assertRepLog(t, 2, append(dummyLog, "03[2,10]"))
}

func TestAppendNewlineWithHashCommand(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", common.DatabasePost{
		Log: dummyLog,
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   2,
				Body: "#flip",
			},
		},
	})
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
		BoardPublic: config.BoardPublic{
			PostParseConfigs: config.PostParseConfigs{
				HashCommands: true,
			},
		},
	})

	sv := newWSServer(t)
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

	if err := appendRune([]byte("10"), cl); err != nil {
		t.Fatal(err)
	}

	t.Run("command type", func(t *testing.T) {
		t.Parallel()

		var typ common.CommandType
		q := db.FindPost(2).Field("commands").AtIndex(0).Field("type")
		if err := db.One(q, &typ); err != nil {
			t.Fatal(err)
		}
		if typ != common.Flip {
			t.Errorf("unexpected command type: %d", typ)
		}
	})

	t.Run("last log message", func(t *testing.T) {
		t.Parallel()

		var log string
		q := db.FindPost(2).Field("log").Nth(-1)
		if err := db.One(q, &log); err != nil {
			t.Fatal(err)
		}
		const std = "03[2,10]"
		if log != std {
			LogUnexpected(t, std, log)
		}
	})

	t.Run("second to last log message", func(t *testing.T) {
		t.Parallel()

		var log string
		q := db.FindPost(2).Field("log").Nth(-2)
		if err := db.One(q, &log); err != nil {
			t.Fatal(err)
		}
		const patt = `09{"id":2,"type":1,"val":(?:true|false)}`
		if !regexp.MustCompile(patt).MatchString(log) {
			t.Fatalf("message does not match `%s`: `%s`", patt, log)
		}
	})
}

func TestAppendNewlineWithLinks(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:   2,
					Body: " >>22 ",
				},
				Board: "a",
				OP:    1,
			},
			Log: []string{},
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 22,
				},
				OP:    21,
				Board: "c",
			},
			Log: []string{},
		},
	})

	sv := newWSServer(t)
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
	setBoardConfigs(t, false)

	if err := appendRune([]byte("10"), cl); err != nil {
		t.Fatal(err)
	}

	std := [...]struct {
		id    uint64
		log   []string
		field string
		val   common.LinkMap
	}{
		{
			id: 2,
			log: []string{
				`07{"id":2,"links":{"22":{"op":21,"board":"c"}}}`,
				`03[2,10]`,
			},
			field: "links",
			val: common.LinkMap{
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
			val: common.LinkMap{
				2: {
					OP:    1,
					Board: "a",
				},
			},
		},
	}

	for i := range std {
		s := std[i]
		t.Run(s.field, func(t *testing.T) {
			t.Parallel()

			assertRepLog(t, s.id, s.log)

			var links common.LinkMap
			q := db.FindPost(s.id).Field(s.field)
			if err := db.One(q, &links); err != nil {
				t.Fatal(err)
			}
			AssertDeepEquals(t, links, s.val)
		})
	}
}

func TestBackspace(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", samplePost)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		time:       time.Now().Unix(),
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	if err := backspace([]byte{}, cl); err != nil {
		t.Fatal(err)
	}

	assertOpenPost(t, cl, 2, "ab")
	assertRepLog(t, 2, append(dummyLog, "042"))
	assertBody(t, 2, "ab")
}

func TestClosePost(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", samplePost)
	setBoardConfigs(t, false)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:         2,
		op:         1,
		bodyLength: 3,
		board:      "a",
		Buffer:     *bytes.NewBuffer([]byte("abc")),
	}

	if err := closePost([]byte{}, cl); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(t, cl.openPost, openPost{})
	assertRepLog(t, 2, append(dummyLog, "062"))
	assertBody(t, 2, "abc")
	assertPostClosed(t, 2)
}

func assertPostClosed(t *testing.T, id uint64) {
	var editing bool
	q := db.FindPost(id).Field("editing")
	if err := db.One(q, &editing); err != nil {
		t.Fatal(err)
	}
	if editing {
		t.Error("post not closed")
	}
}

func TestSpliceValidityChecks(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
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

	cases := [...]struct {
		name       string
		start, len int
		text, line string
		err        error
	}{
		{"negative start", -1, 1, "", "", errInvalidSpliceCoords},
		{"exceeds buffer bounds", 2, 1, "", "abc", errInvalidSpliceCoords},
		{"NOOP", 0, 0, "", "", errSpliceNOOP},
		{"too long", 0, 0, tooLong, "", errSpliceTooLong},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := spliceRequest{
				spliceCoords: spliceCoords{
					Start: c.start,
					Len:   c.len,
				},
				Text: []rune(c.text),
			}
			if err := spliceText(marshalJSON(t, req), cl); err != c.err {
				UnexpectedError(t, err)
			}
		})
	}
}

func TestSplice(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", samplePost)
	setBoardConfigs(t, false)

	const longSplice = `Never gonna give you up ` +
		`Never gonna let you down ` +
		`Never gonna run around and desert you ` +
		`Never gonna make you cry ` +
		`Never gonna say goodbye ` +
		`Never gonna tell a lie and hurt you `

	sv := newWSServer(t)
	defer sv.Close()

	cases := [...]struct {
		name              string
		start, len        int
		text, init, final string
		log               []string
	}{
		{
			name:  "append to empty line",
			start: 0,
			len:   0,
			text:  "abc",
			init:  "",
			final: "abc",
			log:   []string{`05{"id":2,"start":0,"len":0,"text":"abc"}`},
		},
		{
			name:  "remove one char",
			start: 0,
			len:   1,
			text:  "",
			init:  "abc",
			final: "bc",
			log:   []string{`05{"id":2,"start":0,"len":1,"text":""}`},
		},
		{
			name:  "remove one multibyte char",
			start: 2,
			len:   1,
			text:  "",
			init:  "αΒΓΔ",
			final: "αΒΔ",
			log:   []string{`05{"id":2,"start":2,"len":1,"text":""}`},
		},
		{
			name:  "replace till line end",
			start: 2,
			len:   -1,
			text:  "abc",
			init:  "abcd",
			final: "ababc",
			log:   []string{`05{"id":2,"start":2,"len":-1,"text":"abc"}`},
		},
		{
			name:  "replace multibyte char till line end",
			start: 1,
			len:   -1,
			text:  "ΓΔ",
			init:  "αΒΓΔ",
			final: "αΓΔ",
			log:   []string{`05{"id":2,"start":1,"len":-1,"text":"ΓΔ"}`},
		},
		{
			name:  "inject into the middle of the line",
			start: 2,
			len:   -1,
			text:  "abc",
			init:  "ab",
			final: "ababc",
			log:   []string{`05{"id":2,"start":2,"len":-1,"text":"abc"}`},
		},
		{
			name:  "inject multibyte char into the middle of the line",
			start: 2,
			len:   0,
			text:  "Δ",
			init:  "αΒΓ",
			final: "αΒΔΓ",
			log:   []string{`05{"id":2,"start":2,"len":0,"text":"Δ"}`},
		},
		{
			name:  "inject into second line of body",
			start: 2,
			len:   3,
			text:  "abcdefg",
			init:  "00\n012345",
			final: "00\n01abcdefg5",
			log:   []string{`05{"id":2,"start":2,"len":3,"text":"abcdefg"}`},
		},
		{
			name:  "append exceeds max body length",
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
			name:  "injection exceeds max body length",
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
			name:  "splice contains newlines",
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
			name:  "inject single newline char",
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

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			assertTableClear(t, "posts")
			assertInsert(t, "posts", common.DatabasePost{
				StandalonePost: common.StandalonePost{
					Post: common.Post{
						Editing: true,
						ID:      2,
						Body:    c.init,
					},
					Board: "a",
					OP:    1,
				},
				Log: []string{},
			})

			cl, _ := sv.NewClient()
			cl.openPost = openPost{
				id:         2,
				op:         1,
				bodyLength: utf8.RuneCountInString(c.init),
				board:      "a",
				time:       time.Now().Unix(),
				Buffer:     *bytes.NewBufferString(lastLine(c.init)),
			}

			req := spliceRequest{
				spliceCoords: spliceCoords{
					Start: c.start,
					Len:   c.len,
				},
				Text: []rune(c.text),
			}

			if err := spliceText(marshalJSON(t, req), cl); err != nil {
				t.Fatal(err)
			}

			assertOpenPost(
				t,
				cl,
				utf8.RuneCountInString(c.final),
				lastLine(c.final),
			)
			assertBody(t, 2, c.final)
			assertRepLog(t, 2, c.log)
		})
	}
}

func TestCloseOldOpenPost(t *testing.T) {
	assertTableClear(t, "posts")

	then := time.Now().Add(time.Minute * -30).Unix()
	assertInsert(t, "posts", common.DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Editing: true,
				ID:      1,
				Time:    then,
			},
			OP: 1,
		},
		Log: []string{},
	})

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:   1,
		op:   1,
		time: then,
	}

	has, err := cl.hasPost()
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("client has open post")
	}

	var editing bool
	if err := db.One(db.FindPost(1).Field("editing"), &editing); err != nil {
		t.Fatal(err)
	}
	if editing {
		t.Fatal("post not closed")
	}

	assertRepLog(t, 1, []string{"061"})
}

func TestInsertImageIntoPostWithImage(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:       1,
		time:     time.Now().Unix(),
		hasImage: true,
	}
	if err := insertImage(nil, cl); err != errHasImage {
		UnexpectedError(t, err)
	}
}

func TestInsertImageOnTextOnlyBoard(t *testing.T) {
	setBoardConfigs(t, true)

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:    1,
		board: "a",
		time:  time.Now().Unix(),
	}

	req := ImageRequest{
		Name:  "foo.jpeg",
		Token: "123",
	}
	if err := insertImage(marshalJSON(t, req), cl); err != errTextOnly {
		UnexpectedError(t, err)
	}
}

func TestInsertImage(t *testing.T) {
	assertTableClear(t, "posts", "threads", "images", "imageTokens")
	setBoardConfigs(t, false)
	assertInsert(t, "threads", common.DatabaseThread{
		ID:      1,
		Board:   "a",
		PostCtr: 1,
	})
	assertInsert(t, "posts", common.DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID: 2,
			},
			Board: "a",
			OP:    1,
		},
		Log: []string{},
	})
	assertInsert(t, "images", stdJPEG)
	_, token, err := db.NewImageToken(stdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}

	sv := newWSServer(t)
	defer sv.Close()
	cl, _ := sv.NewClient()
	cl.openPost = openPost{
		id:    2,
		board: "a",
		op:    1,
		time:  time.Now().Unix(),
	}

	req := ImageRequest{
		Name:  "foo.jpeg",
		Token: token,
	}
	if err := insertImage(marshalJSON(t, req), cl); err != nil {
		t.Fatal(err)
	}

	std := common.Image{
		Name:        "foo",
		ImageCommon: stdJPEG,
	}
	msg, err := EncodeMessage(MessageInsertImage, imageMessage{
		ID:    2,
		Image: std,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertRepLog(t, 2, []string{string(msg)})
	assertImageCounter(t, 1, 1)

	var res common.Image
	q := db.FindPost(2).Field("image")
	if err := db.One(q, &res); err != nil {
		t.Fatal(err)
	}
	if res != std {
		LogUnexpected(t, std, res)
	}

	if !cl.openPost.hasImage {
		t.Error("no image flag on openPost")
	}
}
