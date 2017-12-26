package imager

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"meguca/db"
	"mime/multipart"
	"runtime"

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
				req := <-requestThumbnailing

				buf := bytes.NewBuffer(thumbnailer.GetBuffer())
				_, err := buf.ReadFrom(req.file)
				data := buf.Bytes()
				if err != nil {
					req.res <- thumbnailingResponse{500, "", err}
					thumbnailer.ReturnBuffer(data)
					continue
				}

				sum := sha1.Sum(data)
				SHA1 := hex.EncodeToString(sum[:])
				img, err := db.GetImage(SHA1)
				var (
					code int
					id   string
				)
				switch err {
				case nil: // Already have a thumbnail
					code, id, err = newImageToken(SHA1)
				case sql.ErrNoRows:
					img.SHA1 = SHA1
					code, id, err = newThumbnail(data, img)
				default:
					code = 500
				}

				req.res <- thumbnailingResponse{code, id, err}
				thumbnailer.ReturnBuffer(data)
			}
		}()
	}
}
