package templates

import (
	"bytes"
	"fmt"
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
	OP             int64
	Board, Subject string
	Lang           lang.Common
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

func wrapPost(
	p common.Post,
	op int64,
	board, subject string,
	lang lang.Common,
) postContext {
	return postContext{
		Post:    p,
		OP:      op,
		Board:   board,
		Subject: subject,
		Lang:    lang,
	}
}

// Returns the HTTP path to the thumbnail of an image
func thumbPath(fileType uint8, SHA1 string) template.HTML {
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

	return template.HTML(buf)
}

// Returns the HTTP path to the source file
func sourcePath(img common.Image) string {
	return fmt.Sprintf(
		"/images/src/%s.%s",
		img.SHA1,
		common.Extensions[img.FileType],
	)
}

func extension(fileType uint8) string {
	return common.Extensions[fileType]
}

// Renders the post creation time field
func formatTime(sec int64, lang map[string][]string) template.HTML {
	t := time.Unix(sec, 0)
	year, m, day := t.Date()
	weekday := lang["week"][int(t.Weekday())]
	month := lang["calendar"][int(m)]

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

	return template.HTML(buf)
}

// Stringify an int and left-pad to at least double digits
func pad(buf []byte, i int) []byte {
	if i < 10 {
		buf = append(buf, '0')
	}
	return append(buf, strconv.Itoa(i)...)
}

// Renders a human-readable representation video/audio length
func readableLength(l uint32) string {
	if l < 60 {
		return fmt.Sprintf("0:%02d", l)
	}
	min := l / 60
	return fmt.Sprintf("%02d:%02d", min, l-min)
}

// Renders a human-readable representation of file size
func readableFileSize(s int) string {
	if s < (1 << 10) {
		return fmt.Sprintf("%d B", s)
	}
	if s < (1 << 20) {
		return fmt.Sprintf("%d KB", s/(1<<10))
	}
	return fmt.Sprintf("%.1f MB", float32(s)/(1<<20))
}

// Render a link to another post. Can optionally be cross-thread.
func renderPostLink(
	id, op int64,
	board, OPLang string,
	cross bool,
) template.HTML {
	var w htmlWriter

	w.WriteString(`<a class="history" href="`)

	// More premature optimization ahead

	// Write href
	if cross {
		w.WriteByte('/')
		w.WriteString(board)
		w.WriteByte('/')
		w.WriteString(strconv.FormatInt(op, 10))
	}
	w.WriteString("#p")
	idStr := strconv.FormatInt(id, 10)
	w.WriteString(idStr)
	w.WriteString(`">>>`)

	// Write text
	if cross {
		w.WriteString(">/")
		w.WriteString(board)
		w.WriteByte('/')
	}
	w.WriteString(idStr)
	if id == op { // OP of this thread
		w.WriteByte(' ')
		w.WriteString(OPLang)
	}

	w.WriteString("</a>")

	return w.HTML()
}
