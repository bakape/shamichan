package imager

import "sync"

var (
	// Pool for thumbnailing operations. Do not use directly!
	_thumbBufPool = sync.Pool{
		New: func() interface{} {
			// Most thumbnails will be bellow or slightly above 8 KB (2 pages),
			// so allocate 3 pages
			return make([]byte, 0, 12<<10)
		},
	}
)

// Get a thumbnailing buffer from the pool
func getThumbBuffer() []byte {
	return _thumbBufPool.Get().([]byte)
}

// Put thumbnailing buffer into the pool
func putThumbBuffer(buf []byte) {
	_thumbBufPool.Put(buf[:0])
}
