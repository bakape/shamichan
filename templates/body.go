package templates

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strconv"

	"github.com/bakape/meguca/config"
)

var (
	commandRegexp   = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount)$`)
	diceRegexp      = regexp.MustCompile(`^(\d*)d(\d+)$`)
	linkRegexp      = regexp.MustCompile(`^>>(>*)(\d+)$`)
	referenceRegexp = regexp.MustCompile(`^>>>(>*)\/(\w+)\/$`)
	urlRegexp       = regexp.MustCompile(
		`^(?:magnet:\?|https?:\/\/)[-a-zA-Z0-9@:%_\+\.~#\?&\/=]+$`,
	)
)

type bodyContext struct {
	state struct { // Body parser state
		spoiler, quote bool
		iDice          int
	}
	postContext
	htmlWriter
}

// Render the text body of a post
func renderBody(p postContext) template.HTML {
	c := bodyContext{
		postContext: p,
	}

	lines := bytes.Split([]byte(c.Body), []byte{'\n'})
	if c.Editing {
		for i := 0; i < len(lines)-1; i++ {
			c.parseTerminatedLine(lines[i])
		}
		c.parseOpenLine(lines[len(lines)-1])
	} else {
		for _, line := range lines {
			c.parseTerminatedLine(line)
		}
	}
	return c.HTML()
}

// Parse a line that is no longer being edited
func (c *bodyContext) parseTerminatedLine(line []byte) {
	// For hiding redundant newlines using CSS
	if len(line) == 0 {
		c.WriteString("<br>")
		return
	}

	c.initLine(line[0])

	if line[0] == '#' {
		if m := commandRegexp.FindSubmatch(line); m != nil {
			c.parseCommands(string(m[1]))
			c.terminateTags(true)
			return
		}
	}

	c.parseSpoilers(line, (*c).parseFragment)
	c.terminateTags(true)
}

// Open a new line container and check for quotes
func (c *bodyContext) initLine(first byte) {
	c.state.spoiler = false
	c.state.quote = false

	c.WriteString("<span>")
	if first == '>' {
		c.WriteString("<em>")
		c.state.quote = true
	}
}

// Injects spoiler tags and calls fn on the remaining parts
func (c *bodyContext) parseSpoilers(frag []byte, fn func([]byte)) {
	for {
		i := bytes.Index(frag, []byte{'*', '*'})
		if i != -1 {
			fn(frag[:i])
			if c.state.spoiler {
				c.WriteString("</del>")
			} else {
				c.WriteString("<del>")
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
func (c *bodyContext) parseFragment(frag []byte) {
	for i, word := range bytes.Split(frag, []byte{' '}) {
		if i != 0 {
			c.WriteByte(' ')
		}
		if len(word) == 0 {
			continue
		}
		if word[0] == '>' {
			if m := linkRegexp.FindSubmatch(word); m != nil {
				// Post links
				c.parsePostLink(m)
				continue
			} else if m := referenceRegexp.FindSubmatch(word); m != nil {
				// Internal and custom reference URLs
				c.parseReference(m)
				continue
			}
		} else {
			// Generic HTTP(S) URLs and magnet links
			match := false
			// Checking the first byte is much cheaper than a function call. Do
			// that first, as most cases won't match.
			switch word[0] {
			case 'h':
				match = bytes.HasPrefix(word, []byte("http"))
			case 'm':
				match = bytes.HasPrefix(word, []byte("magnet:?"))
			}
			if match {
				c.parseURL(word)
				continue
			}
		}
		c.escape(word)
	}
}

// Parse a potential link to a post
func (c *bodyContext) parsePostLink(m [][]byte) {
	if c.Links == nil {
		c.Write(m[0])
		return
	}

	id, _ := strconv.ParseUint(string(m[2]), 10, 64)
	verified, ok := c.Links[id]
	if !ok {
		c.Write(m[0])
		return
	}

	if len(m[1]) != 0 { // Write extra quotes
		c.Write(m[1])
	}
	html := renderPostLink(
		id,
		verified.OP,
		verified.Board,
		c.Lang.Posts["OP"],
		verified.OP != c.OP,
	)
	c.WriteString(string(html))
}

// Parse internal or customly set reference URL
func (c *bodyContext) parseReference(m [][]byte) {
	var (
		m2   = string(m[2])
		href string
	)
	if config.IsBoard(m2) {
		href = fmt.Sprintf("/%s/", m2)
	} else if href = config.Get().Links[m2]; href != "" {
	} else {
		c.Write(m[0])
		return
	}

	if len(m[1]) != 0 {
		c.Write(m[1])
	}
	c.newTabLink(href, fmt.Sprintf(">>>/%s/", string(m[2])))
}

// Format and anchor link that opens in a new tab
func (c *bodyContext) newTabLink(href, text string) {
	fmt.Fprintf(
		c,
		`<a href="%s" target="_blank">%s</a>`,
		url.QueryEscape(href),
		html.EscapeString(text),
	)
}

// Parse generic URLs and magnet links
func (c *bodyContext) parseURL(bit []byte) {
	s := string(bit)
	switch {
	case !urlRegexp.Match(bit):
		c.escape(bit)
	case bit[0] == 'm': // Don't open a new tab for magnet links
		fmt.Fprintf(
			c,
			`<a href="%s">%s</a>`,
			url.QueryEscape(s),
			html.EscapeString(s),
		)
	default:
		c.newTabLink(s, s)
	}
}

// Write an HTML-escaped string to buffer
func (c *bodyContext) escape(bit []byte) {
	c.WriteString(html.EscapeString(string(bit)))
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

	fmt.Fprintf(c, "<strong>#%s (%s)</strong>", bit, inner.String())
}

// If command validation failed, simply write the string
func (c *bodyContext) writeInvalidCommand(bit string) {
	c.WriteByte('#')
	c.WriteString(bit)
}

// Close any open HTML tags
func (c *bodyContext) terminateTags(newLine bool) {
	if c.state.spoiler {
		c.WriteString("</del>")
	}
	if c.state.quote {
		c.WriteString("</em>")
	}
	if newLine {
		c.WriteString("<br>")
	}
	c.WriteString("</span>")
}

// Parse a line that is still being edited
func (c *bodyContext) parseOpenLine(line []byte) {
	if len(line) == 0 {
		c.WriteString("<span></span>")
		return
	}
	c.initLine(line[0])
	c.parseSpoilers(line, (*c).escape)
	c.terminateTags(false)
}
