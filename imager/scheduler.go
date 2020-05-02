package imager

import (
	"crypto/sha1"
	"hash"
	"io"
	"mime/multipart"
	"runtime"
)

var (
	_scheduleJob      = make(chan jobRequest, 128)
	_scheduleSmallJob = make(chan jobRequest, 128)
)

type thumbnailingRequest struct {
	insertionRequest
	file multipart.File
	size int
}

type jobRequest struct {
	thumbnailingRequest
	res chan<- error
}

// Queues upload processing to prevent resource overuse.
// Takes ownership of `req.file`.
func requestThumbnailing(req thumbnailingRequest) <-chan error {
	// 2 separate queues - one for small and one for bigger files.
	// Allows for some degree of concurrent thumbnailing without exhausting
	// server resources.
	ch := make(chan error)
	jReq := jobRequest{req, ch}
	if req.size <= 4<<20 {
		_scheduleSmallJob <- jReq
	} else {
		_scheduleJob <- jReq
	}
	return ch
}

// Queue thumbnailing jobs to reduce resource contention and prevent OOM
func init() {
	for _, ch := range [...]<-chan jobRequest{_scheduleJob, _scheduleSmallJob} {
		go func(queue <-chan jobRequest) {
			// Prevents needless spawning of more threads by the Go runtime
			runtime.LockOSThread()

			for req := range queue {
				// Check, if client still there, before and after thumbnailing
				select {
				case <-req.ctx.Done():
				default:
					select {
					case <-req.ctx.Done():
					case req.res <- processRequest(req.thumbnailingRequest):
					}
				}

				// Always deallocate file in the same spot to not cause
				// data races
				req.file.Close()
			}
		}(ch)
	}
}

// Hash file from disk in 4KB chunks
func hashFile(
	dst []byte,
	rs io.ReadSeeker,
	h hash.Hash,
) (
	read int,
	err error,
) {
	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}

	var (
		arr [4 << 10]byte
		buf []byte
		n   int
	)
	for {
		n, err = rs.Read(arr[:])
		buf = arr[:n]
		read += n
		switch err {
		case nil:
			h.Write(buf)
		case io.EOF:
			err = nil
			copy(dst, h.Sum(buf))
			return
		default:
			return
		}
	}
}

func processRequest(req thumbnailingRequest) (err error) {
	var id [20]byte
	_, err = hashFile(id[:], req.file, sha1.New())
	if err != nil {
		return
	}

	err = tryInsertExisting(req.insertionRequest, id)
	if err != errNotProcessed {
		return
	}
	return insertNewThumbnail(req, id)
}
