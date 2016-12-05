package templates

import (
	"bytes"
	"html"
	"html/template"
	"net/url"
	"strconv"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/lang"
)

type htmlWriter struct {
	bytes.Buffer
}

// Allows passing additional information to thread-related templates
type postContext struct {
	common.Post
	OP              uint64
	Omit, ImageOmit int
	Subject, Root   string
}

// Write an element attribute to the buffer
func (w *htmlWriter) attr(key, val string) {
	w.WriteByte(' ')
	w.WriteString(key)
	if val != "" {
		w.WriteString(`="`)
		w.WriteString(val)
		w.WriteByte('"')
	}
}

// Write an HTML-escaped string to buffer
func (w *htmlWriter) escape(s string) {
	w.WriteString(html.EscapeString(s))
}

// Write an URL-query-escaped string to buffer
func (w *htmlWriter) queryEscape(s string) {
	w.WriteString(url.QueryEscape(s))
}

// Outputs the buffer contents as a HTML string
func (w *htmlWriter) HTML() template.HTML {
	return template.HTML(w.String())
}

// Returns the HTTP path to the thumbnail of an image
func thumbPath(fileType uint8, SHA1 string) string {
	buf := make([]byte, 14, 58)
	copy(buf, "/images/thumb/")
	buf = append(buf, SHA1...)
	buf = append(buf, '.')

	var ext string
	if fileType == common.JPEG {
		ext = "jpg"
	} else {
		ext = "png"
	}
	buf = append(buf, ext...)

	return string(buf)
}

// Returns the HTTP path to the source file
func sourcePath(fileType uint8, SHA1 string) string {
	ext := common.Extensions[fileType]

	buf := make([]byte, 12, 53+len(ext))
	copy(buf, "/images/src/")
	buf = append(buf, SHA1...)
	buf = append(buf, '.')
	buf = append(buf, ext...)

	return string(buf)
}

// Returns image name with proper extension
func imageName(fileType uint8, name string) string {
	ext := common.Extensions[fileType]
	l := len(name)
	buf := make([]byte, l, l+1+len(ext))
	copy(buf, name)
	buf = append(buf, '.')
	return html.EscapeString(string(append(buf, ext...)))
}

// Renders the post creation time field
func formatTime(sec int64) string {
	ln := lang.Packs["en_GB"].Common.Time

	t := time.Unix(sec, 0)
	year, m, day := t.Date()
	weekday := ln["week"][int(t.Weekday())]
	// Months are 1-indexed for some fucking reason
	month := ln["calendar"][int(m)-1]

	// Premature optimization
	buf := make([]byte, 0, 17+len(weekday)+len(month))
	buf = pad(buf, day)
	buf = append(buf, ' ')
	buf = append(buf, month...)
	buf = append(buf, ' ')
	buf = append(buf, strconv.Itoa(year)...)
	buf = append(buf, " ("...)
	buf = append(buf, weekday...)
	buf = append(buf, ") "...)
	buf = pad(buf, t.Hour())
	buf = append(buf, ':')
	buf = pad(buf, t.Minute())

	return string(buf)
}

// Stringify an int and left-pad to at least double digits
func pad(buf []byte, i int) []byte {
	if i < 10 {
		buf = append(buf, '0')
	}
	return append(buf, strconv.Itoa(i)...)
}

// Formats a human-readable representation of file size
func readableFileSize(s int) string {
	format := func(n, end string) string {
		l := len(n)
		buf := make([]byte, l, l+len(end))
		copy(buf, n)
		return string(append(buf, end...))
	}

	switch {
	case s < (1 << 10):
		return format(strconv.Itoa(s), " B")
	case s < (1 << 20):
		return format(strconv.Itoa(s/(1<<10)), " KB")
	default:
		n := strconv.FormatFloat(float64(s)/(1<<20), 'f', 1, 32)
		return format(n, " MB")
	}
}

// Render a link to another post. Can optionally be cross-thread.
func renderPostLink(id, op uint64, board string, cross bool) string {
	var w htmlWriter

	w.WriteString(`<a class="history" data-id=`)
	idStr := strconv.FormatUint(id, 10)
	w.WriteString(idStr)
	w.WriteString(` href="`)

	// More premature optimization ahead

	// Write href
	if cross {
		w.WriteByte('/')
		w.WriteString(board)
		w.WriteByte('/')
		w.WriteString(strconv.FormatUint(op, 10))
	}
	w.WriteString("#p")
	w.WriteString(idStr)
	w.WriteString(`">>>`)

	// Write text
	if cross {
		w.WriteString(">/")
		w.WriteString(board)
		w.WriteByte('/')
	}
	w.WriteString(idStr)

	w.WriteString("</a>")

	return w.String()
}

// Correct thumbnail dimensions for smaller reply thumbnails
func correctDims(large bool, w, h uint16) (string, string) {
	if !large && (w > 125 || h > 125) {
		w = uint16(float32(w) * 0.8333)
		h = uint16(float32(h) * 0.8333)
	}
	return strconv.FormatUint(uint64(w), 10), strconv.FormatUint(uint64(h), 10)
}
