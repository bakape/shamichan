package imager

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"meguca/db"
	"mime/multipart"

	"github.com/bakape/thumbnailer"
)

var (
	scheduleJob = make(chan jobRequest)
	errTimedOut = errors.New("thumbnailing timed out")
)

type jobRequest struct {
	file multipart.File
	res  chan<- thumbnailingResponse
}

type thumbnailingResponse struct {
	code    int
	imageID string
	err     error
}

// Queues larger uplaod processing to prevent resource overuse
func requestThumbnailing(
	file multipart.File,
	size int64,
) <-chan thumbnailingResponse {
	ch := make(chan thumbnailingResponse)

	// Small uploads can be scheduled to their own goroutine concurently without
	// much resource contention
	if size <= smallUploadSize {
		go func() {
			code, id, err := processRequest(file)
			ch <- thumbnailingResponse{code, id, err}
		}()
	} else {
		scheduleJob <- jobRequest{file, ch}
	}
	return ch
}

// Queue larger thumbnailing jobs to reduce resource contention
func init() {
	go func() {
		for {
			req := <-scheduleJob
			code, id, err := processRequest(req.file)
			req.res <- thumbnailingResponse{code, id, err}
		}
	}()
}

func processRequest(file multipart.File) (code int, id string, err error) {
	buf := bytes.NewBuffer(thumbnailer.GetBuffer())
	_, err = buf.ReadFrom(file)
	data := buf.Bytes()
	defer thumbnailer.ReturnBuffer(data)
	if err != nil {
		code = 500
		return
	}

	sum := sha1.Sum(data)
	SHA1 := hex.EncodeToString(sum[:])
	img, err := db.GetImage(SHA1)
	switch err {
	case nil: // Already have a thumbnail
		return newImageToken(SHA1)
	case sql.ErrNoRows:
		img.SHA1 = SHA1
		return newThumbnail(data, img)
	default:
		code = 500
		return
	}
}
