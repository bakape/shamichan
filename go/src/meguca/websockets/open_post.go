package websockets

import (
	"meguca/common"
	"unicode/utf8"
)

// Data of a post currently being written to by a Client
type openPost struct {
	hasImage, isSpoilered bool
	len, lines            int
	id, op                uint64
	time                  int64
	body                  []byte
	board                 string
}

// Initialize a new open post from a post struct
func (o *openPost) init(p common.StandalonePost) {
	*o = openPost{
		id:    p.ID,
		op:    p.OP,
		time:  p.Time,
		board: p.Board,
		len:   utf8.RuneCountInString(p.Body),
		body:  append(make([]byte, 0, 1<<10), p.Body...),
	}
	o.countLines()
	if p.Image != nil {
		o.hasImage = true
		o.isSpoilered = p.Image.Spoiler
	}
}

// Count amount of lines in the post body
func (o *openPost) countLines() {
	o.lines = 0
	for _, b := range o.body {
		if b == '\n' {
			o.lines++
		}
	}
}
