package imager

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"meguca/common"
	"meguca/db"
	"mime/multipart"

	"github.com/bakape/thumbnailer"
)

var (
	scheduleJob = make(chan jobRequest, 128)
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
