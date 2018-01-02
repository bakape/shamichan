package imager

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"meguca/common"
	"meguca/db"
	"mime/multipart"
	"runtime"
	"time"

	"github.com/bakape/thumbnailer"
)

// Balances thumbnailing across worker threads to prevent resource overuse
var requestThumbnailing chan thumbnailingRequest

type thumbnailingRequest struct {
	file multipart.File
	res  chan<- thumbnailingResponse
}

type thumbnailingResponse struct {
	code    int
	imageID string
	err     error
}

// Spawn thumbnailing workers
func init() {
	n := runtime.NumCPU() - 1
	if n < 1 {
		n = 1
	}
	requestThumbnailing = make(chan thumbnailingRequest, n)
	for i := 0; i < n; i++ {
		go func() {
			for {
				// Perform thumbnailing in separate goroutine, so we can time
				// out the request after 10 seconds. Unsure why this happens,
				// but lengthy requests should not block workers.
				req := <-requestThumbnailing
				to := time.After(time.Second * 10)
				ch := make(chan thumbnailingResponse, 1)
				go processRequest(req.file, ch)
				select {
				case res := <-ch:
					req.res <- res
				case <-to:
					req.res <- thumbnailingResponse{
						500,
						"",
						errors.New("thumbnailing timed out"),
					}
				}
			}
		}()
	}
}

func processRequest(file multipart.File, ch chan<- thumbnailingResponse) {
	var res thumbnailingResponse

	buf := bytes.NewBuffer(thumbnailer.GetBuffer())
	_, res.err = buf.ReadFrom(file)
	data := buf.Bytes()
	defer thumbnailer.ReturnBuffer(data)
	if res.err != nil {
		res.code = 500
		ch <- res
		return
	}

	sum := sha1.Sum(data)
	SHA1 := hex.EncodeToString(sum[:])
	var img common.ImageCommon
	img, res.err = db.GetImage(SHA1)
	switch res.err {
	case nil: // Already have a thumbnail
		res.code, res.imageID, res.err = newImageToken(SHA1)
	case sql.ErrNoRows:
		img.SHA1 = SHA1
		res.code, res.imageID, res.err = newThumbnail(data, img)
	default:
		res.code = 500
	}
	ch <- res
}
