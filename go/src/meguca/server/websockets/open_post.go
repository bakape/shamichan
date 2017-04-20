package websockets

// Data of a post currently being written to by a Client
type openPost struct {
	hasImage, isSpoilered bool
	len, lines            int
	id, op                uint64
	time                  int64
	body                  []byte
	board                 string
}
