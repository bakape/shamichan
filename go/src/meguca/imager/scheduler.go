package imager

import (
	"bytes"
	"container/list"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"meguca/db"
	"mime/multipart"
	"runtime"
	"time"

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

// Balances thumbnailing across worker threads to prevent resource overuse
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

// Schedule larger thumbnailing jobs to reduce resource contention
func init() {
	go func() {
		var (
			waiting     = list.New()
			canSchedule = runtime.NumCPU() + 1
			done        = make(chan bool)
		)

		doJob := func(req jobRequest) {
			// Perform thumbnailing in separate goroutine, so we can time
			// out the request after 10 seconds. Unsure why this happens,
			// but lengthy requests should not block workers.
			to := time.NewTimer(time.Second * 10)

			// Buffer to prevent timed out goroutines from leaking
			ch := make(chan thumbnailingResponse, 1)

			go func() {
				code, id, err := processRequest(req.file)
				ch <- thumbnailingResponse{code, id, err}
			}()
			go func() {
				select {
				case res := <-ch:
					req.res <- res
				case <-to.C:
					req.res <- thumbnailingResponse{500, "", errTimedOut}
				}
				to.Stop()
				done <- true
			}()
		}

		for {
			select {
			case req := <-scheduleJob:
				if canSchedule == 0 {
					waiting.PushBack(req)
				} else {
					canSchedule--
					doJob(req)
				}
			case <-done:
				canSchedule++
				if waiting.Len() != 0 {
					canSchedule--
					doJob(waiting.Remove(waiting.Front()).(jobRequest))
				}
			}
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
