package imager

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"meguca/common"
	"meguca/db"
	"mime/multipart"

	"github.com/bakape/thumbnailer"
)

// Size of an upload small enough to use a different processing priority
const smallUploadSize = 4 << 20

var (
	scheduleJob = make(chan jobRequest)
	errTimedOut = errors.New("thumbnailing timed out")
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

// Queues larger uplaod processing to prevent resource overuse
func requestThumbnailing(file multipart.File, size int,
) <-chan thumbnailingResponse {
	ch := make(chan thumbnailingResponse)

	// Small uploads can be scheduled to their own goroutine concurrently
	// without much resource contention
	if size <= smallUploadSize {
		go func() {
			id, err := processRequest(file, size)
			ch <- thumbnailingResponse{id, err}
		}()
	} else {
		scheduleJob <- jobRequest{file, size, ch}
	}
	return ch
}

// Queue larger thumbnailing jobs to reduce resource contention
func init() {
	go func() {
		for {
			req := <-scheduleJob
			id, err := processRequest(req.file, req.size)
			req.res <- thumbnailingResponse{id, err}
		}
	}()
}

func processRequest(file multipart.File, size int) (string, error) {
	data := thumbnailer.GetBufferCap(size)
	data, err := thumbnailer.ReadInto(data, file)
	if err != nil {
		return "", common.StatusError{err, 500}
	}
	defer thumbnailer.ReturnBuffer(data)
	if err != nil {
		return "", common.StatusError{err, 500}
	}

	sum := sha1.Sum(data)
	SHA1 := hex.EncodeToString(sum[:])
	img, err := db.GetImage(SHA1)
	switch err {
	case nil: // Already have a thumbnail
		return db.NewImageToken(SHA1)
	case sql.ErrNoRows:
		img.SHA1 = SHA1
		return newThumbnail(data, img)
	default:
		return "", common.StatusError{err, 500}
	}
}
