// Package imager handles image, video, etc. upload requests and processing
package imager

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"errors"
	"image"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/thumbnailer/v2"
	"github.com/chai2010/webp"
	"github.com/go-playground/log"
)

// Minimal capacity of large buffers in the pool
const largeBufCap = 12 << 10

var (
	// Map of MIME types to the constants used internally
	mimeTypes = map[string]uint8{
		"image/jpeg":                    common.JPEG,
		"image/png":                     common.PNG,
		"image/gif":                     common.GIF,
		"image/webp":                    common.WEBP,
		mimePDF:                         common.PDF,
		"video/webm":                    common.WEBM,
		"application/ogg":               common.OGG,
		"video/mp4":                     common.MP4,
		"video/quicktime":               common.MP4,
		"audio/mpeg":                    common.MP3,
		mime7Zip:                        common.SevenZip,
		mimeTarGZ:                       common.TGZ,
		mimeTarXZ:                       common.TXZ,
		mimeZip:                         common.ZIP,
		"audio/x-flac":                  common.FLAC,
		mimeText:                        common.TXT,
		"application/x-rar-compressed":  common.RAR,
		"application/vnd.comicbook+zip": common.CBZ,
		"application/vnd.comicbook-rar": common.CBR,
	}

	// MIME types from thumbnailer to accept
	allowedMimeTypes map[string]bool

	errTooLarge    = errors.New("file too large")
	errSecretImage = errors.New("image secret post detected")

	// Large buffer pool of length=0 capacity=12+KB
	largeBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, largeBufCap)
		},
	}
)

func init() {
	allowedMimeTypes = make(map[string]bool, len(mimeTypes))
	for t := range mimeTypes {
		allowedMimeTypes[t] = true
	}
}

// Return large buffer pool, if eligable
func returnLargeBuf(buf []byte) {
	if cap(buf) >= largeBufCap {
		largeBufPool.Put(buf[:0])
	}
}

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(w http.ResponseWriter, r *http.Request) {
	var id string
	err := func() (err error) {
		err = validateUploader(w, r)
		if err != nil {
			return
		}

		// Limit data received to the maximum uploaded file size limit
		r.Body = http.MaxBytesReader(w, r.Body, int64(config.Get().MaxSize<<20))

		id, err = ParseUpload(r)
		if err != nil {
			return
		}
		return incrementSpamScore(w, r)
	}()
	if err != nil {
		LogError(w, r, err)
	}

	w.Write([]byte(id))
}

// Apply security restrictions to uploader
func validateUploader(w http.ResponseWriter, r *http.Request) (err error) {
	if s := r.Header.Get("Authorization"); s != "" &&
		s == "Bearer "+config.Get().Salt {
		// Internal upload bypass
		return nil
	}

	ip, err := auth.GetIP(r)
	if err != nil {
		return
	}
	err = db.IsBanned("all", ip)
	if err != nil {
		return
	}

	var session auth.Base64Token
	err = session.EnsureCookie(w, r)
	if err != nil {
		return
	}
	need, err := db.NeedCaptcha(session, ip)
	if err != nil {
		return
	}
	if need {
		return common.StatusError{
			Err:  errors.New("captcha required"),
			Code: 403,
		}
	}

	return
}

// UploadImageHash attempts to skip image upload, if the file has already
// been thumbnailed and is stored on the server. The client sends an SHA1 hash
// of the file it wants to upload. The server looks up, if such a file is
// thumbnailed. If yes, generates and sends a new image allocation token to
// the client.
func UploadImageHash(w http.ResponseWriter, r *http.Request) {
	token, err := func() (token string, err error) {
		err = validateUploader(w, r)
		if err != nil {
			return
		}

		buf, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 40))
		if err != nil {
			return
		}
		sha1 := string(buf)

		err = db.InTransaction(func(tx *pgx.Tx) (err error) {
			exists, err := db.ImageExists(tx, sha1)
			if err != nil {
				return
			}
			if exists {
				token, err = db.NewImageToken(tx, sha1)
			}
			return
		})
		if err != nil {
			return
		}
		err = incrementSpamScore(w, r)
		return
	}()
	if err != nil {
		LogError(w, r, err)
	} else if token != "" {
		w.Write([]byte(token))
	}
}

func incrementSpamScore(w http.ResponseWriter, r *http.Request) (err error) {
	ip, err := auth.GetIP(r)
	if err != nil {
		return
	}
	var session auth.Base64Token
	err = session.EnsureCookie(w, r)
	if err != nil {
		return
	}
	db.IncrementSpamScore(session, ip, config.Get().ImageScore)
	return
}

// LogError send the client file upload errors and logs them server-side
func LogError(w http.ResponseWriter, r *http.Request, err error) {
	code := 500
	if err, ok := err.(common.StatusError); ok {
		code = err.Code
	}
	http.Error(w, err.Error(), code)

	if common.IsTest || common.CanIgnoreClientError(err) {
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
func ParseUpload(req *http.Request) (id string, err error) {
	max := config.Get().MaxSize << 20
	length, err := strconv.ParseUint(req.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		err = common.StatusError{
			Err:  err,
			Code: 413,
		}
		return
	}
	if uint(length) > max {
		err = common.StatusError{
			Err:  errTooLarge,
			Code: 400,
		}
		return
	}
	err = req.ParseMultipartForm(0)
	if err != nil {
		err = common.StatusError{
			Err:  err,
			Code: 400,
		}
		return
	}

	file, head, err := req.FormFile("image")
	if err != nil {
		err = common.StatusError{
			Err:  err,
			Code: 400,
		}
		return
	}
	defer file.Close()
	if uint(head.Size) > max {
		err = common.StatusError{
			Err:  errTooLarge,
			Code: 413,
		}
		return
	}

	// Reject image sekritpost
	var (
		buf    [6]byte
		secret = [6]byte{'s', 'e', 'c', 'r', 'e', 't'}
		mathes = true
	)
	file.ReadAt(buf[:], head.Size-6) // Ignore read errors here
	for i, b := range buf {
		// Lowecase all ASCII
		if 'A' <= b && b <= 'Z' {
			b += 'a' - 'A'
		}
		if b != secret[i] {
			mathes = false
			break
		}
	}
	if mathes {
		err = common.StatusError{
			Err:  errSecretImage,
			Code: 400,
		}
		return
	}

	res := <-requestThumbnailing(file, int(head.Size))
	return res.imageID, res.err
}

// Create a new thumbnail, commit its resources to the DB and filesystem, and
// pass the image data to the client.
func newThumbnail(f multipart.File, SHA1 string) (
	token string, err error,
) {
	var img common.ImageCommon
	img.SHA1 = SHA1

	conf := config.Get()
	thumb, err := processFile(f, &img, thumbnailer.Options{
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
	defer returnLargeBuf(thumb)
	if err != nil {
		switch err.(type) {
		case thumbnailer.ErrUnsupportedMIME, thumbnailer.ErrInvalidImage:
			err = common.StatusError{
				Err:  err,
				Code: 400,
			}
		}
		return
	}

	// Being done in one transaction prevents the image DB record from getting
	// garbage-collected between the calls
	err = db.InTransaction(func(tx *pgx.Tx) (err error) {
		var thumbR io.ReadSeeker
		if thumb != nil {
			thumbR = bytes.NewReader(thumb)
		}
		err = db.AllocateImage(tx, f, thumbR, img)
		if err != nil && !db.IsConflictError(err) {
			return
		}
		token, err = db.NewImageToken(tx, img.SHA1)
		return
	})
	return
}

// Separate function for easier testability
func processFile(f multipart.File, img *common.ImageCommon,
	opts thumbnailer.Options,
) (
	thumb []byte, err error,
) {
	src, thumbImage, err := thumbnailer.Process(f, opts)
	defer func() {
		// Add image internal buffer to pool
		if thumbImage == nil {
			return
		}
		// Only image type used in thumbnailer by default
		img, ok := thumbImage.(*image.RGBA)
		if ok {
			returnLargeBuf(img.Pix)
		}
	}()
	switch err {
	case nil:
		img.ThumbType = common.WEBP
	case thumbnailer.ErrCantThumbnail:
		err = nil
		img.ThumbType = common.NoFile
	default:
		return
	}

	img.FileType = mimeTypes[src.Mime]

	img.Audio = src.HasAudio
	img.Video = src.HasVideo
	img.Length = uint32(src.Length / time.Second)

	// Some media has retardedly long meta strings. Just truncate them, instead
	// of rejecting.
	img.Artist = src.Artist
	img.Title = src.Title
	if len(img.Artist) > 100 {
		img.Artist = img.Artist[:100]
	}
	if len(img.Title) > 200 {
		img.Title = img.Title[:200]
	}

	img.Dims = [4]uint16{uint16(src.Width), uint16(src.Height), 0, 0}
	if thumbImage != nil {
		b := thumbImage.Bounds()
		img.Dims[2] = uint16(b.Dx())
		img.Dims[3] = uint16(b.Dy())
	}

	img.MD5, img.Size, err = hashFile(f, md5.New(),
		base64.RawURLEncoding.EncodeToString)
	if err != nil {
		return
	}

	if thumbImage != nil {
		w := bytes.NewBuffer(largeBufPool.Get().([]byte))
		err = webp.Encode(w, thumbImage, &webp.Options{
			Lossless: false,
			Quality:  90,
		})
		if err != nil {
			return
		}
		thumb = w.Bytes()
	}

	return
}
