package templates

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
	"github.com/valyala/quicktemplate"
)

func TestRenderBody(t *testing.T) {
	config.Set(config.Configs{
		Public: config.Public{
			Links: map[string]string{
				"4chan": "http://4chan.org",
			},
		},
	})
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

	cases := [...]struct {
		name, in, out, string string
		editing               bool
		op                    uint64
		links                 [][2]uint64
		commands              []common.Command
	}{
		{
			name: "closed post",
			in:   "foo\nbar",
			out:  "foo<br>bar",
		},
		{
			name:    "open post",
			in:      "foo\nbar",
			out:     "foo<br>bar",
			editing: true,
		},
		{
			name: "closed post quote",
			in:   ">foo\nbar",
			out:  "<em>&gt;foo</em><br>bar",
		},
		{
			name:    "open post quote",
			in:      ">foo\nbar",
			out:     "<em>&gt;foo</em><br>bar",
			editing: true,
		},
		{
			name: "closed post spoiler",
			in:   "foo**bar** baz",
			out:  "foo<del>bar</del> baz",
		},
		{
			name:    "open post spoiler",
			in:      "foo**bar** baz",
			out:     "foo<del>bar</del> baz",
			editing: true,
		},
		{
			name: "hide empty lines",
			in:   "bar\n\n",
			out:  "bar<br><br>",
		},
		{
			name: "unclosed spoiler tags",
			in:   "**foo",
			out:  "<del>foo</del>",
		},
		{
			name:    "trailing empty open line",
			in:      "foo\n",
			out:     "foo<br>",
			editing: true,
		},
		{
			name: "#flip",
			in:   "#flip\n#flip",
			out:  "<strong>#flip (true)</strong><br><strong>#flip (false)</strong>",
			commands: []common.Command{
				{
					Type: common.Flip,
					Val:  true,
				},
				{
					Type: common.Flip,
					Val:  false,
				},
			},
		},
		{
			name: "#8ball",
			in:   "#8ball",
			out:  "<strong>#8ball (bar)</strong>",
			commands: []common.Command{
				{
					Type: common.EightBall,
					Val:  "bar",
				},
			},
		},
		{
			name: "#pyu",
			in:   "#pyu",
			out:  "<strong>#pyu (1)</strong>",
			commands: []common.Command{
				{
					Type: common.Pyu,
					Val:  int64(1),
				},
			},
		},
		{
			name: "#pcount",
			in:   "#pcount",
			out:  "<strong>#pcount (2)</strong>",
			commands: []common.Command{
				{
					Type: common.Pcount,
					Val:  int64(2),
				},
			},
		},
		{
			name: "single roll dice",
			in:   "#d20",
			out:  "<strong>#d20 (22)</strong>",
			commands: []common.Command{
				{
					Type: common.Dice,
					// This is how values are decoded from the database
					Val: []interface{}{float64(22)},
				},
			},
		},
		{
			name: "multiple roll dice",
			in:   "#2d20",
			out:  "<strong>#2d20 (22 + 33 = 55)</strong>",
			commands: []common.Command{
				{
					Type: common.Dice,
					Val:  []interface{}{float64(22), float64(33)},
				},
			},
		},
		{
			name: "too many dice rolls",
			in:   "#11d20",
			out:  "#11d20",
			commands: []common.Command{
				{
					Type: common.Dice,
					Val:  []interface{}{float64(22), float64(33)},
				},
			},
		},
		{
			name: "too many dice faces",
			in:   "#2d101",
			out:  "#2d101",
			commands: []common.Command{
				{
					Type: common.Dice,
					Val:  []interface{}{float64(22), float64(33)},
				},
			},
		},
		{
			name: "no valid commands",
			in:   "#flip",
			out:  "#flip",
		},
		{
			name: "too few commands",
			in:   "#flip\n#flip",
			out:  "<strong>#flip (true)</strong><br>#flip",
			commands: []common.Command{
				{
					Type: common.Flip,
					Val:  true,
				},
			},
		},
		{
			name: "no links in post",
			in:   ">>20",
			out:  "<em>>>20</em>",
		},
		{
			name:  "1 invalid link",
			in:    ">>20",
			out:   "<em>>>20</em>",
			links: [][2]uint64{{21, 21}},
		},
		{
			name:  "valid link",
			in:    ">>21",
			out:   `<em><a class="history post-link" data-id="21" href="#p21">>>21</a><a class="hash-link history" href="#p21"> #</a></em>`,
			op:    20,
			links: [][2]uint64{{21, 20}},
		},
		{
			name:  "valid link with extra quotes",
			in:    ">>>>21",
			out:   `<em>>><a class="history post-link" data-id="21" href="#p21">>>21</a><a class="hash-link history" href="#p21"> #</a></em>`,
			op:    20,
			links: [][2]uint64{{21, 20}},
		},
		{
			name:  "valid cross-thread link",
			in:    ">>21",
			out:   `<em><a class="history post-link" data-id="21" href="/all/22#p21">>>21 ➡</a><a class="hash-link history" href="/all/22#p21"> #</a></em>`,
			op:    20,
			links: [][2]uint64{{21, 22}},
		},
		{
			name: "invalid reference",
			in:   ">>>/fufufu/",
			out:  `<em>>>>/fufufu/</em>`,
		},
		{
			name: "link reference",
			in:   ">>>/4chan/",
			out:  `<em><a href="http://4chan.org" target="_blank">&gt;&gt;&gt;/4chan/</a></em>`,
		},
		{
			name: "board reference",
			in:   ">>>/a/",
			out:  `<em><a href="/a/" target="_blank">&gt;&gt;&gt;/a/</a></em>`,
		},
		{
			name: "reference with extra quotes",
			in:   ">>>>>/a/",
			out:  `<em>>><a href="/a/" target="_blank">&gt;&gt;&gt;/a/</a></em>`,
		},
		{
			name: "HTTP URL",
			in:   "http://4chan.org",
			out:  `<a href="http://4chan.org" target="_blank">http://4chan.org</a>`,
		},
		{
			name: "HTTPS URL",
			in:   "https://4chan.org",
			out:  `<a href="https://4chan.org" target="_blank">https://4chan.org</a>`,
		},
		{
			name: "magnet URL",
			in:   "magnet:?xt=urn:btih:c12fe1",
			out:  `<a href="magnet:?xt=urn:btih:c12fe1">magnet:?xt=urn:btih:c12fe1</a>`,
		},
		{
			name: "XSS inject URL",
			in:   "http://4chan.org<>",
			out:  `http://4chan.org&lt;&gt;`,
		},
		{
			name: "escape generic text",
			in:   "<>&",
			out:  "&lt;&gt;&amp;",
		},
		{
			name: "youtube embed",
			in:   "https://www.youtube.com/watch?v=z0f4Wgi94eo",
			out:  "<em><a class=\"embed\" target=\"_blank\" data-type=\"0\" href=\"https://www.youtube.com/watch?v=z0f4Wgi94eo\">[Youtube] ???</a></em>",
		},
		{
			name: "youtu.be embed",
			in:   "https://youtu.be/z0f4Wgi94eo",
			out:  "<em><a class=\"embed\" target=\"_blank\" data-type=\"0\" href=\"https://youtu.be/z0f4Wgi94eo\">[Youtube] ???</a></em>",
		},
		{
			name: "soundcloud embed",
			in:   "https://soundcloud.com/cd_oblongar",
			out:  "<em><a class=\"embed\" target=\"_blank\" data-type=\"1\" href=\"https://soundcloud.com/cd_oblongar\">[SoundCloud] ???</a></em>",
		},
		{
			name: "vimeo embed",
			in:   "https://vimeo.com/174312494",
			out:  "<em><a class=\"embed\" target=\"_blank\" data-type=\"2\" href=\"https://vimeo.com/174312494\">[Vimeo] ???</a></em>",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			p := common.Post{
				Body:     c.in,
				Editing:  c.editing,
				Links:    c.links,
				Commands: c.commands,
			}

			buf := quicktemplate.AcquireByteBuffer()
			defer quicktemplate.ReleaseByteBuffer(buf)
			w := quicktemplate.AcquireWriter(buf)
			defer quicktemplate.ReleaseWriter(w)

			streambody(w, p, c.op, false)

			if s := string(buf.B); s != c.out {
				LogUnexpected(t, c.out, s)
			}
		})
	}
}
