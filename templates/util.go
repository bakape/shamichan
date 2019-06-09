package templates

import (
	"html"
)

func bold(s string) string {
	s = html.EscapeString(s)
	b := make([]byte, 3, len(s)+7)
	copy(b, "<b>")
	b = append(b, s...)
	b = append(b, "</b>"...)
	return string(b)
}
