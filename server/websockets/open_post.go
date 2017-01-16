package websockets

import "bytes"

// Data of a post currently being written to by a Client
type openPost struct {
	hasImage bool
	bytes.Buffer
	len    int
	id, op uint64
	time   int64
	board  string
}

func (o *openPost) LastLine() []byte {
	b := o.Bytes()
	i := bytes.LastIndexByte(b, '\n')
	if i == -1 {
		return b
	}
	return b[i+1:]
}

func (o *openPost) TrimLastLine() {
	b := o.Bytes()
	o.Truncate(bytes.LastIndexByte(b, '\n') + 1)
}
