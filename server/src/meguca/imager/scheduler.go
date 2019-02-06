package imager

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"hash"
	"io"
	"meguca/common"
	"meguca/db"
	"mime/multipart"
	"sync"
)

var (
	scheduleJob = make(chan jobRequest, 128)

	// Pool of temp buffers used for hashing
	buf512Pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 512)
		},
	}
)

type jobRequest struct {
	file multipart.File
	size int
	res  chan<- thumbnailingResponse
}

type thumbnailingResponse struct {
	imageID string
	err     error
}

// Queues upload processing to prevent resource overuse
func requestThumbnailing(file multipart.File, size int,
) <-chan thumbnailingResponse {
	ch := make(chan thumbnailingResponse)
	scheduleJob <- jobRequest{file, size, ch}
	return ch
}

// Queue thumbnailing jobs to reduce resource contention and prevent OOM
func init() {
	go func() {
		for {
			req := <-scheduleJob
			id, err := processRequest(req.file, req.size)
			req.res <- thumbnailingResponse{id, err}
		}
	}()
}

// Hash file to string
func hashFile(rs io.ReadSeeker, h hash.Hash, encode func([]byte) string,
) (
	hash string, read int, err error,
) {

	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}
	buf := buf512Pool.Get().([]byte)
	defer buf512Pool.Put(buf)

	for {
		buf = buf[:512] // Reset slicing

		var n int
		n, err = rs.Read(buf)
		buf = buf[:n]
		read += n
		switch err {
		case nil:
			h.Write(buf)
		case io.EOF:
			err = nil
			hash = encode(h.Sum(buf))
			return
		default:
			return
		}
	}
}

func processRequest(file multipart.File, size int) (token string, err error) {
	SHA1, _, err := hashFile(file, sha1.New(), hex.EncodeToString)
	if err != nil {
		return
	}
	img, err := db.GetImage(SHA1)
	switch err {
	case nil: // Already have a thumbnail
		return db.NewImageToken(SHA1)
	case sql.ErrNoRows:
		img.SHA1 = SHA1
		return newThumbnail(file, img)
	default:
		err = common.StatusError{err, 500}
		return
	}
}
