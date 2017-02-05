package templates

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/valyala/quicktemplate"
)

// Embeddable URL types
const (
	youTube = iota
	soundCloud
	vimeo
)

var (
	commandRegexp   = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount)$`)
	diceRegexp      = regexp.MustCompile(`^(\d*)d(\d+)$`)
	linkRegexp      = regexp.MustCompile(`^>>(>*)(\d+)$`)
	referenceRegexp = regexp.MustCompile(`^>>>(>*)\/(\w+)\/$`)
	urlRegexp       = regexp.MustCompile(`^(?:magnet:\?|https?:\/\/)[-a-zA-Z0-9@:%_\+\.~#\?&\/=]+$`)

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
)

type bodyContext struct {
	state struct { // Body parser state
		spoiler, quote, lastLineEmpty bool
		iDice                         int
	}
	common.Post
	OP uint64
	quicktemplate.Writer
}

// Render the text body of a post
func streambody(w *quicktemplate.Writer, p common.Post, op uint64) {
	c := bodyContext{
		Post:   p,
		OP:     op,
		Writer: *w,
	}

	var fn func(string)
	if c.Editing {
		fn = c.parseOpenLine
	} else {
		fn = c.parseTerminatedLine
	}

	lines := strings.Split(c.Body, "\n")
	last := len(lines) - 1
	for i, l := range lines {
		// Prevent successive empty lines
		if len(l) == 0 {
			// Don't break, if body ends with newline
			if !c.state.lastLineEmpty && i != last {
				c.N().S("<br>")
			}
			c.state.lastLineEmpty = true
			c.state.quote = false
			continue
		}
		c.state.lastLineEmpty = false

		c.initLine(l[0])
		fn(l)
		c.terminateTags(i != last)
	}
}

// Parse a line that is no longer being edited
func (c *bodyContext) parseTerminatedLine(line string) {
	c.parseSpoilers(line, (*c).parseFragment)
}

// Open a new line container and check for quotes
func (c *bodyContext) initLine(first byte) {
	c.state.quote = false
	c.state.lastLineEmpty = false
	if first == '>' {
		c.N().S("<em>")
		c.state.quote = true
	}
	if c.state.spoiler {
		c.N().S("<del>")
	}
}

// Injects spoiler tags and calls fn on the remaining parts
func (c *bodyContext) parseSpoilers(frag string, fn func(string)) {
	for {
		i := strings.Index(frag, "**")
		if i != -1 {
			fn(frag[:i])
			if c.state.quote {
				c.N().S("</em>")
			}
			if c.state.spoiler {
				c.N().S("</del>")
			} else {
				c.N().S("<del>")
			}
			if c.state.quote {
				c.N().S("<em>")
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
	for i, word := range strings.Split(frag, " ") {
		if i != 0 {
			c.N().S(` `)
		}
		if len(word) == 0 {
			continue
		}
		switch word[0] {
		case '>': // Links
			if m := linkRegexp.FindStringSubmatch(word); m != nil {
				// Post links
				c.parsePostLink(m)
				continue
			} else if m := referenceRegexp.FindStringSubmatch(word); m != nil {
				// Internal and custom reference URLs
				c.parseReference(m)
				continue
			}
		case '#': // Hash commands
			if m := commandRegexp.FindStringSubmatch(word); m != nil {
				c.parseCommands(string(m[1]))
				continue
			}
		default: // Generic HTTP(S) URLs and magnet links
			match := false
			// Checking the first byte is much cheaper than a function call. Do
			// that first, as most cases won't match.
			switch word[0] {
			case 'h':
				match = strings.HasPrefix(word, "http")
			case 'm':
				match = strings.HasPrefix(word, "magnet:?")
			}
			if match {
				c.parseURL(word)
				continue
			}
		}
		c.E().S(word)
	}
}

// Parse a potential link to a post
func (c *bodyContext) parsePostLink(m []string) {
	if c.Links == nil {
		c.N().S(m[0])
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
		c.N().S(m[0])
		return
	}

	if len(m[1]) != 0 { // Write extra quotes
		c.N().S(m[1])
	}
	streampostLink(&c.Writer, id, op, op != c.OP)
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
		c.N().S(m[0])
		return
	}

	if len(m[1]) != 0 {
		c.N().S(m[1])
	}
	c.newTabLink(href, fmt.Sprintf(">>>/%s/", string(m[2])))
}

// Format and anchor link that opens in a new tab
func (c *bodyContext) newTabLink(href, text string) {
	c.N().S(`<a href="`)
	c.E().S(href)
	c.N().S(`" target="_blank">`)
	c.E().S(text)
	c.N().S(`</a>`)
}

// Parse generic URLs and magnet links
func (c *bodyContext) parseURL(bit string) {
	s := string(bit)
	switch {
	case !urlRegexp.MatchString(bit):
		c.E().S(bit)
	case c.parseEmbeds(bit):
	case bit[0] == 'm': // Don't open a new tab for magnet links
		s = html.EscapeString(s)
		c.N().S(`<a href="`)
		c.N().S(s)
		c.N().S(`">`)
		c.N().S(s)
		c.N().S(`</a>`)
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

		c.N().S(`<em><a class="embed" target="_blank" data-type="`)
		c.N().D(t.typ)
		c.N().S(`" href="`)
		c.E().S(s)
		c.N().S(`">[`)
		c.N().S(providers[t.typ])
		c.N().S(`] ???</a></em>`)

		return true
	}
	return false
}

// Parse a hash command
func (c *bodyContext) parseCommands(bit string) {
	// Guard against invalid dice rolls
	invalid := c.Commands == nil ||
		c.state.iDice > len(c.Commands)-1 ||
		c.Commands[c.state.iDice].Val == nil
	if invalid {
		c.writeInvalidCommand(bit)
		return
	}

	// TODO: Sycnwatch

	inner := new(bytes.Buffer)
	switch bit {
	case "flip", "8ball", "pyu", "pcount":
		fmt.Fprint(inner, c.Commands[c.state.iDice].Val)
		c.state.iDice++
	default:
		// Validate dice
		m := diceRegexp.FindStringSubmatch(bit)
		if m[1] != "" {
			if rolls, err := strconv.Atoi(m[1]); err != nil || rolls > 10 {
				c.writeInvalidCommand(bit)
				return
			}
		}
		if sides, err := strconv.Atoi(m[2]); err != nil || sides > 100 {
			c.writeInvalidCommand(bit)
			return
		}

		// Cast []interface to []uint16
		uncast := c.Commands[c.state.iDice].Val.([]interface{})
		rolls := make([]uint16, len(uncast))
		for i := range rolls {
			rolls[i] = uint16(uncast[i].(float64))
		}

		c.state.iDice++
		var sum uint
		for i, roll := range rolls {
			if i != 0 {
				inner.WriteString(" + ")
			}
			sum += uint(roll)
			inner.WriteString(strconv.FormatUint(uint64(roll), 10))
		}
		if len(rolls) > 1 {
			fmt.Fprintf(inner, " = %d", sum)
		}
	}

	c.N().S(`<strong>#`)
	c.N().S(bit)
	c.N().S(` (`)
	c.N().S(inner.String())
	c.N().S(`)</strong>`)
}

// If command validation failed, simply write the string
func (c *bodyContext) writeInvalidCommand(s string) {
	c.N().S("#")
	c.N().S(s)
}

// Close any open HTML tags
func (c *bodyContext) terminateTags(newLine bool) {
	if c.state.spoiler {
		c.N().S("</del>")
	}
	if c.state.quote {
		c.N().S("</em>")
	}
	if newLine {
		c.N().S("<br>")
	}
}

// Parse a line that is still being edited
func (c *bodyContext) parseOpenLine(line string) {
	c.parseSpoilers(line, func(s string) {
		c.E().S(s)
	})
}
