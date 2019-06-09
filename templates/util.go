package templates

import (
	"html"

	"github.com/bakape/meguca/common"
)

// CalculateOmit returns the omitted post and image counts for a thread
func CalculateOmit(t common.Thread) (int, int) {
	var (
		omit    = int(t.PostCount) - (len(t.Posts) + 1)
		imgOmit uint32
	)
	if omit != 0 {
		imgOmit = t.ImageCount
		if t.Image != nil {
			imgOmit--
		}
		for _, p := range t.Posts {
			if p.Image != nil {
				imgOmit--
			}
		}
	}
	return omit, int(imgOmit)
}

func bold(s string) string {
	s = html.EscapeString(s)
	b := make([]byte, 3, len(s)+7)
	copy(b, "<b>")
	b = append(b, s...)
	b = append(b, "</b>"...)
	return string(b)
}
