// Package imager handles image, video, etc. upload requests and processing
package imager

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"net/http"
	"strconv"
	"time"

	"github.com/Soreil/apngdetector"
	"github.com/bakape/thumbnailer"
	"github.com/go-playground/log"
)

var (
	// Map of MIME types to the constants used internally
	mimeTypes = map[string]uint8{
		"image/jpeg":      common.JPEG,
		"image/png":       common.PNG,
		"image/gif":       common.GIF,
		mimePDF:           common.PDF,
		"video/webm":      common.WEBM,
		"application/ogg": common.OGG,
		"video/mp4":       common.MP4,
		"video/quicktime": common.MP4,
		"audio/mpeg":      common.MP3,
		mime7Zip:          common.SevenZip,
		mimeTarGZ:         common.TGZ,
		mimeTarXZ:         common.TXZ,
		mimeZip:           common.ZIP,
		"audio/x-flac":    common.FLAC,
		mimeText:          common.TXT,
	}

	// MIME types from thumbnailer to accept
	allowedMimeTypes = map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/gif":       true,
		"application/pdf": true,
		"video/webm":      true,
		"application/ogg": true,
		"video/mp4":       true,
		"video/quicktime": true,
		"audio/mpeg":      true,
		mimeZip:           true,
		mime7Zip:          true,
		mimeTarGZ:         true,
		mimeTarXZ:         true,
		"audio/x-flac":    true,
		mimeText:          true,
	}

	errTooLarge = errors.New("file too large")
	isTest      bool
)

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(w http.ResponseWriter, r *http.Request) {
	var id string
	err := func() (err error) {
		err = validateUploader(r)
		if err != nil {
			return
		}

		// Limit data received to the maximum uploaded file size limit
		r.Body = http.MaxBytesReader(w, r.Body, int64(config.Get().MaxSize<<20))

		id, err = ParseUpload(r)
		return
	}()
	if err != nil {
		LogError(w, r, err)
	}

	w.Write([]byte(id))
}

// Apply security restrictions to uploader
// TODO: Needs to consider spam score
func validateUploader(r *http.Request) (err error) {
	ip, err := auth.GetIP(r)
	if err != nil {
		return
	}
	err = db.IsBanned("all", ip)
	return
}

// UploadImageHash attempts to skip image upload, if the file has already
// been thumbnailed and is stored on the server. The client sends an SHA1 hash
// of the file it wants to upload. The server looks up, if such a file is
// thumbnailed. If yes, generates and sends a new image allocation token to
// the client.
func UploadImageHash(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		err = validateUploader(r)
		if err != nil {
			return
		}

		buf, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 40))
		if err != nil {
			return
		}
		hash := string(buf)

		_, err = db.GetImage(hash)
		switch err {
		case nil:
		case sql.ErrNoRows:
			return nil
		default:
			return
		}

		token, err := db.NewImageToken(hash)
		if err != nil {
			return
		}
		w.Write([]byte(token))
		return
	}()
	if err != nil {
		LogError(w, r, err)
	}
}

// LogError send the client file upload errors and logs them server-side
func LogError(w http.ResponseWriter, r *http.Request, err error) {
	code := 500
	if err, ok := err.(common.StatusError); ok {
		code = err.Code
	}
	http.Error(w, err.Error(), code)

	if isTest || common.CanIgnoreClientError(err) {
		return
	}
	ip, ipErr := auth.GetIP(r)
	if ipErr != nil {
		ip = "invalid IP"
	}
	log.Errorf("upload error: by %s: %s: %#v", ip, err, err)
}

// ParseUpload parses the upload form. Separate function for cleaner error
// handling and reusability.
// Returns the HTTP status code of the response, the ID of the generated image
// and an error, if any.
func ParseUpload(req *http.Request) (string, error) {
	max := config.Get().MaxSize << 20
	length, err := strconv.ParseUint(req.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return "", common.StatusError{err, 413}
	}
	if uint(length) > max {
		return "", common.StatusError{errTooLarge, 400}
	}
	err = req.ParseMultipartForm(0)
	if err != nil {
		return "", common.StatusError{err, 400}
	}

	file, head, err := req.FormFile("image")
	if err != nil {
		return "", common.StatusError{err, 400}
	}
	if uint(head.Size) > max {
		return "", common.StatusError{errTooLarge, 413}
	}
	defer file.Close()
	res := <-requestThumbnailing(file, int(head.Size))
	return res.imageID, res.err
}

// Create a new thumbnail, commit its resources to the DB and filesystem, and
// pass the image data to the client.
func newThumbnail(data []byte, img common.ImageCommon,
) (token string, err error) {
	conf := config.Get()
	thumb, err := processFile(data, &img, thumbnailer.Options{
		JPEGQuality: conf.JPEGQuality,
		MaxSourceDims: thumbnailer.Dims{
			Width:  uint(conf.MaxWidth),
			Height: uint(conf.MaxHeight),
		},
		ThumbDims: thumbnailer.Dims{
			Width:  150,
			Height: 150,
		},
		AcceptedMimeTypes: allowedMimeTypes,
	})
	if err != nil {
		return "", WrapThumbnailerError(err)
	}
	defer func() {
		if thumb != nil {
			thumbnailer.ReturnBuffer(thumb)
		}
	}()

	// Some media has retardedly long meta strings. Just truncate them, instead
	// of rejecting.
	if len(img.Artist) > 100 {
		img.Artist = img.Artist[:100]
	}
	if len(img.Title) > 200 {
		img.Title = img.Title[:200]
	}

	// Being done in one transaction prevents the image from getting
	// garbage-collected between the calls
	err = db.InTransaction(false, func(tx *sql.Tx) (err error) {
		err = db.AllocateImage(tx, data, thumb, img)
		if err != nil && !db.IsConflictError(err) {
			return
		}
		token, err = db.NewImageTokenTx(tx, img.SHA1)
		return
	})
	return
}

// WrapThumbnailerError Wraps a thumbnailer error with the appropriate HTTP status code
func WrapThumbnailerError(err error) error {
	switch err.(type) {
	case nil:
		return nil
	case thumbnailer.ErrUnsupportedMIME, thumbnailer.ErrInvalidImage,
		thumbnailer.ErrCorruptImage:
		return common.StatusError{err, 400}
	default:
		return common.StatusError{err, 500}
	}
}

// Separate function for easier testability
func processFile(
	data []byte,
	img *common.ImageCommon,
	opts thumbnailer.Options,
) (
	thumbData []byte,
	err error,
) {
	src, thumb, err := thumbnailer.ProcessBuffer(data, opts)
	switch err {
	case nil:
	case thumbnailer.ErrNoCoverArt:
		err = nil
	default:
		return
	}

	thumbData = thumb.Data

	img.FileType = mimeTypes[src.Mime]
	if img.FileType == common.PNG {
		img.APNG = apngdetector.Detect(data)
	}
	if thumb.Data == nil {
		img.ThumbType = common.NoFile
	} else if thumb.IsPNG {
		img.ThumbType = common.PNG
	}

	img.Audio = src.HasAudio
	img.Video = src.HasVideo
	img.Length = uint32(src.Length / time.Second)
	img.Size = len(data)
	img.Artist = src.Artist
	img.Title = src.Title
	img.Dims = [4]uint16{
		uint16(src.Width),
		uint16(src.Height),
		uint16(thumb.Width),
		uint16(thumb.Height),
	}

	sum := md5.Sum(data)
	img.MD5 = base64.RawURLEncoding.EncodeToString(sum[:])

	return
}
