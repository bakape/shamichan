// Package imager handles image, video, etc. upload requests and processing
package imager

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"log"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"net/http"
	"strconv"
	"time"

	"github.com/Soreil/apngdetector"
	"github.com/bakape/thumbnailer"
)

var (
	// Map of MIME types to the constants used internally
	mimeTypes = map[string]uint8{
		"image/jpeg":      common.JPEG,
		"image/png":       common.PNG,
		"image/gif":       common.GIF,
		"application/pdf": common.PDF,
		"video/webm":      common.WEBM,
		"application/ogg": common.OGG,
		"video/mp4":       common.MP4,
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
	// Limit data received to the maximum uploaded file size limit
	r.Body = http.MaxBytesReader(w, r.Body, int64(config.Get().MaxSize<<20))

	code, id, err := ParseUpload(r)
	if err != nil {
		LogError(w, r, code, err)
	}
	w.Write([]byte(id))
}

// UploadImageHash attempts to skip image upload, if the file has already
// been thumbnailed and is stored on the server. The client sends an SHA1 hash
// of the file it wants to upload. The server looks up, if such a file is
// thumbnailed. If yes, generates and sends a new image allocation token to
// the client.
func UploadImageHash(w http.ResponseWriter, req *http.Request) {
	buf, err := ioutil.ReadAll(http.MaxBytesReader(w, req.Body, 40))
	if err != nil {
		LogError(w, req, 500, err)
		return
	}
	hash := string(buf)

	_, err = db.GetImage(hash)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return
	default:
		LogError(w, req, 500, err)
		return
	}

	token, err := db.NewImageToken(hash)
	if err != nil {
		LogError(w, req, 500, err)
	}
	w.Write([]byte(token))
}

// LogError send the client file upload errors and logs them server-side
func LogError(w http.ResponseWriter, r *http.Request, code int, err error) {
	text := err.Error()
	http.Error(w, text, code)
	if !isTest {
		ip, err := auth.GetIP(r)
		if err != nil {
			ip = "invalid IP"
		}
		log.Printf("upload error: %s: %s\n", ip, text)
	}
}

// ParseUpload parses the upload form. Separate function for cleaner error
// handling and reusability. Returns the HTTP status code of the response and an
// error, if any.
func ParseUpload(req *http.Request) (int, string, error) {
	if err := parseUploadForm(req); err != nil {
		return 400, "", err
	}

	file, _, err := req.FormFile("image")
	if err != nil {
		return 400, "", err
	}
	defer file.Close()

	buf := bytes.NewBuffer(thumbnailer.GetBuffer())
	_, err = buf.ReadFrom(file)
	if err != nil {
		return 500, "", err
	}
	data := buf.Bytes()
	defer thumbnailer.ReturnBuffer(data)

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
		return 500, "", err
	}
}

func newImageToken(SHA1 string) (int, string, error) {
	token, err := db.NewImageToken(SHA1)
	code := 200
	if err != nil {
		code = 500
	}
	return code, token, err
}

// Parse and validate the form of the upload request
func parseUploadForm(req *http.Request) error {
	length, err := strconv.ParseUint(req.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}
	if length > uint64(config.Get().MaxSize<<20) {
		return errTooLarge
	}
	return req.ParseMultipartForm(0)
}

// Create a new thumbnail, commit its resources to the DB and filesystem, and
// pass the image data to the client.
func newThumbnail(data []byte, img common.ImageCommon) (int, string, error) {
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
	switch err.(type) {
	case nil:
	case thumbnailer.UnsupportedMIMEError:
		return 400, "", err
	default:
		return 500, "", err
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

	if err := db.AllocateImage(data, thumb, img); err != nil {
		return 500, "", err
	}
	return newImageToken(img.SHA1)
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
