package templates

import (
	"fmt"
	"html"
	"meguca/common"
	"meguca/config"
	"meguca/util"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/valyala/quicktemplate"
)

// Embeddable URL types
const (
	youTube = iota
	soundCloud
	vimeo
)

var (
	linkRegexp      = regexp.MustCompile(`^>>(>*)(\d+)$`)
	referenceRegexp = regexp.MustCompile(`^>>>(>*)\/(\w+)\/$`)
	dickRegex       = regexp.MustCompile(`(?i)(dick|cock)s?`)

	providers = map[int]string{
		youTube:    "Youtube",
		soundCloud: "SoundCloud",
		vimeo:      "Vimeo",
	}
	embedPatterns = [...]struct {
		typ  int
		patt *regexp.Regexp
	}{
		{
			youTube,
			regexp.MustCompile(`https?:\/\/(?:[^\.]+\.)?youtube\.com\/watch\/?\?(?:.+&)?v=([^&]+)`),
		},
		{
			youTube,
			regexp.MustCompile(`https?:\/\/(?:[^\.]+\.)?(?:youtu\.be|youtube\.com\/embed)\/([a-zA-Z0-9_-]+)`),
		},
		{
			soundCloud,
			regexp.MustCompile(`https?:\/\/soundcloud.com\/.*`),
		},
		{
			vimeo,
			regexp.MustCompile(`https?:\/\/(?:www\.)?vimeo\.com\/.+`),
		},
	}

	// URLs supported for linkification
	urlPrefixes = map[byte]string{
		'h': "http",
		'm': "magnet:?",
		'i': "irc",
		'f': "ftp",
		'b': "bitcoin",
	}
)

type bodyContext struct {
	index bool     // Rendered for an index page
	state struct { // Body parser state
		spoiler, quote, code bool
		newlines             uint
		iDice                int
	}
	common.Post
	OP    uint64
	board string
	quicktemplate.Writer
}

// Render the text body of a post
func streambody(
	w *quicktemplate.Writer,
	p common.Post,
	op uint64,
	board string,
	index bool,
) {
	c := bodyContext{
		index:  index,
		Post:   p,
		OP:     op,
		board:  board,
		Writer: *w,
	}

	var fn func(string)
	if c.Editing {
		fn = c.parseOpenLine
	} else {
		fn = c.parseTerminatedLine
	}

	for i, l := range strings.Split(c.Body, "\n") {
		c.state.quote = false

		// Prevent successive empty lines
		if i != 0 && c.state.newlines < 2 {
			c.string("<br>")
		}
		if len(l) == 0 {
			c.state.newlines++
			continue
		}

		c.state.newlines = 0
		if l[0] == '>' {
			c.string("<em>")
			c.state.quote = true
		}
		if c.state.spoiler {
			c.string("<del>")
		}

		fn(l)

		if c.state.spoiler {
			c.string("</del>")
		}
		if c.state.quote {
			c.string("</em>")
		}
	}
}

// Write string without escaping
func (c *bodyContext) string(s string) {
	c.N().S(s)
}

// Escape and write string
func (c *bodyContext) escape(s string) {
	c.E().S(s)
}

// Write a byte without heap allocations or escaping
func (c *bodyContext) byte(b byte) {
	buf := [1]byte{b}
	c.N().SZ(buf[:])
}

// Parse a line that is no longer being edited
func (c *bodyContext) parseTerminatedLine(line string) {
	c.parseCode(line, (*c).parseFragment)
}

// Detect code tags
func (c *bodyContext) parseCode(frag string, fn func(string)) {
	for {
		i := strings.Index(frag, "``")
		if i != -1 {
			c.formatCode(frag[:i], fn)
			frag = frag[i+2:]
			c.state.code = !c.state.code
		} else {
			c.formatCode(frag, fn)
			break
		}
	}
}

func (c *bodyContext) formatCode(frag string, fn func(string)) {
	if c.state.code {
		// Strip quotes
		for len(frag) != 0 && frag[0] == '>' {
			c.string(`&gt;`)
			frag = frag[1:]
		}
		c.N().Z(highlightSyntax(frag))
	} else {
		c.parseSpoilers(frag, fn)
	}
}

// Injects spoiler tags and calls fn on the remaining parts
func (c *bodyContext) parseSpoilers(frag string, fn func(string)) {
	for {
		i := strings.Index(frag, "**")
		if i != -1 {
			fn(frag[:i])
			if c.state.quote {
				c.string("</em>")
			}
			if c.state.spoiler {
				c.string("</del>")
			} else {
				c.string("<del>")
			}
			if c.state.quote {
				c.string("<em>")
			}

			c.state.spoiler = !c.state.spoiler
			frag = frag[i+2:]
		} else {
			fn(frag)
			break
		}
	}
}

// Parse a line fragment
func (c *bodyContext) parseFragment(frag string) {
	// Leading and trailing punctuation, if any
	var leadPunct, trailPunct byte

	for i, word := range strings.Split(frag, " ") {
		if i != 0 {
			c.byte(' ')
		}

		// Strip leading and trailing punctuation and commit separately
		leadPunct, word, trailPunct = util.SplitPunctuationString(word)
		if leadPunct != 0 {
			c.byte(leadPunct)
		}

		if len(word) == 0 {
			goto end
		}
		switch word[0] {
		case '>': // Links
			if m := linkRegexp.FindStringSubmatch(word); m != nil {
				// Post links
				c.parsePostLink(m)
				goto end
			} else if m := referenceRegexp.FindStringSubmatch(word); m != nil {
				// Internal and custom reference URLs
				c.parseReference(m)
				goto end
			}
		case '#': // Hash commands
			if m := common.CommandRegexp.FindStringSubmatch(word); m != nil {
				c.parseCommands(string(m[1]))
				goto end
			}
		default: // Generic HTTP(S) URLs and magnet links
			// Checking the first byte is much cheaper than a function call. Do
			// that first, as most cases won't match.
			pre, ok := urlPrefixes[word[0]]
			if ok && strings.HasPrefix(word, pre) {
				c.parseURL(word)
				goto end
			}
		}

		if c.board == "a" && dickRegex.MatchString(word) {
			if unicode.IsUpper(rune(word[0])) {
				word = "Privilege"
			} else {
				word = "privilege"
			}
		}
		c.escape(word)

	end:
		// Write trailing punctuation, if any
		if trailPunct != 0 {
			c.byte(trailPunct)
		}
	}
}

// Parse a potential link to a post
func (c *bodyContext) parsePostLink(m []string) {
	if c.Links == nil {
		c.string(m[0])
		return
	}

	id, _ := strconv.ParseUint(string(m[2]), 10, 64)
	var op uint64
	for _, l := range c.Links {
		if l[0] == id {
			op = l[1]
			break
		}
	}
	if op == 0 {
		c.string(m[0])
		return
	}

	if len(m[1]) != 0 { // Write extra quotes
		c.string(m[1])
	}
	streampostLink(&c.Writer, id, op != c.OP, c.index)
}

// Parse internal or customly set reference URL
func (c *bodyContext) parseReference(m []string) {
	var (
		m2   = string(m[2])
		href string
	)
	if config.IsBoard(m2) {
		href = fmt.Sprintf("/%s/", m2)
	} else if href = config.Get().Links[m2]; href != "" {
	} else {
		c.string(m[0])
		return
	}

	if len(m[1]) != 0 {
		c.string(m[1])
	}
	c.newTabLink(href, fmt.Sprintf(">>>/%s/", string(m[2])))
}

// Format and anchor link that opens in a new tab
func (c *bodyContext) newTabLink(href, text string) {
	c.string(`<a rel="noreferrer" href="`)
	c.escape(href)
	c.string(`" target="_blank">`)
	c.escape(text)
	c.string(`</a>`)
}

// Parse generic URLs and magnet links
func (c *bodyContext) parseURL(bit string) {
	s := string(bit)
	u, err := url.Parse(s)
	switch {
	case err != nil || u.Path == s: // Invalid or empty path
		c.escape(bit)
	case c.parseEmbeds(bit):
	case bit[0] == 'm': // Don't open a new tab for magnet links
		s = html.EscapeString(s)
		c.string(`<a rel="noreferrer" href="`)
		c.string(s)
		c.string(`">`)
		c.string(s)
		c.string(`</a>`)
	default:
		c.newTabLink(s, s)
	}
}

// Parse select embeddable URLs. Returns, if any found.
func (c *bodyContext) parseEmbeds(s string) bool {
	for _, t := range embedPatterns {
		if !t.patt.MatchString(s) {
			continue
		}

		c.string(`<em><a rel="noreferrer" class="embed" target="_blank" data-type="`)
		c.N().D(t.typ)
		c.string(`" href="`)
		c.escape(s)
		c.string(`">[`)
		c.string(providers[t.typ])
		c.string(`] ???</a></em>`)

		return true
	}
	return false
}

// Parse a hash command
func (c *bodyContext) parseCommands(bit string) {
	// Guard against invalid dice rolls
	if c.Commands == nil || c.state.iDice > len(c.Commands)-1 {
		c.writeInvalidCommand(bit)
		return
	}

	inner := make([]byte, 0, 32)
	val := c.Commands[c.state.iDice]
	switch bit {
	case "flip":
		var s string
		if val.Flip {
			s = "flap"
		} else {
			s = "flop"
		}
		inner = append(inner, s...)
		c.state.iDice++
	case "8ball":
		inner = append(inner, val.Eightball...)
		c.state.iDice++
	case "pyu", "pcount":
		inner = strconv.AppendUint(inner, val.Pyu, 10)
		c.state.iDice++
	default:
		if strings.HasPrefix(bit, "sw") {
			c.formatSyncwatch(val.SyncWatch)
			c.state.iDice++
			return
		}

		// Validate dice
		m := common.DiceRegexp.FindStringSubmatch(bit)
		if m[1] != "" {
			if rolls, err := strconv.Atoi(m[1]); err != nil || rolls > 10 {
				c.writeInvalidCommand(bit)
				return
			}
		}
		sides, err := strconv.Atoi(m[2])
		if err != nil || sides > common.MaxDiceSides {
			c.writeInvalidCommand(bit)
			return
		}

		c.state.iDice++
		var sum uint64
		for i, roll := range val.Dice {
			if i != 0 {
				inner = append(inner, " + "...)
			}
			sum += uint64(roll)
			inner = strconv.AppendUint(inner, uint64(roll), 10)
		}
		if len(val.Dice) > 1 {
			inner = append(inner, " = "...)
			inner = strconv.AppendUint(inner, sum, 10)
		}
	}

	c.string(`<strong>#`)
	c.string(bit)
	c.string(` (`)
	c.N().Z(inner)
	c.string(`)</strong>`)
}

// Format a synchronized time counter
func (c *bodyContext) formatSyncwatch(val [5]uint64) {
	c.string(`<em><strong class="embed syncwatch" data-hour=`)
	c.uint64(val[0])
	c.string(` data-min=`)
	c.uint64(val[1])
	c.string(` data-sec=`)
	c.uint64(val[2])
	c.string(` data-start=`)
	c.uint64(val[3])
	c.string(` data-end=`)
	c.uint64(val[4])
	c.string(`>syncwatch</strong></em>`)
}

func (c *bodyContext) uint64(i uint64) {
	c.string(strconv.FormatUint(i, 10))
}

// If command validation failed, simply write the string
func (c *bodyContext) writeInvalidCommand(s string) {
	c.byte('#')
	c.escape(s)
}

// Parse a line that is still being edited
func (c *bodyContext) parseOpenLine(line string) {
	c.parseCode(line, func(s string) {
		c.escape(s)
	})
}
