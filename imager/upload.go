// Package imager handles image, video, etc. upload requests and processing
package imager

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/websockets"
	"github.com/bakape/thumbnailer/v2"
	"github.com/chai2010/webp"
	"github.com/go-playground/log"
	"github.com/jackc/pgx/v4"
)

var (
	// Map of MIME types to the constants used internally
	mimeTypes = map[string]common.FileType{
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

	errTooLarge = common.StatusError{
		Err:  errors.New("file too large"),
		Code: 400,
	}
	errNoCandidatePost = common.StatusError{
		Err:  errors.New("no open post without image"),
		Code: 404,
	}
	errNotProcessed = common.StatusError{
		Err:  errors.New("hash not in database"),
		Code: 404,
	}
)

func init() {
	allowedMimeTypes = make(map[string]bool, len(mimeTypes))
	for t := range mimeTypes {
		allowedMimeTypes[t] = true
	}
}

// Request to insert an image into the client's open post
type insertionRequest struct {
	Spoiler bool
	Name    string
	ctx     context.Context
	user    auth.AuthKey
}

// Handles the clients' image (or other file) upload request
func NewImageUpload(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, func() (err error) {
		var req insertionRequest
		req.ctx = r.Context()

		req.user, err = validateUploader(w, r)
		if err != nil {
			return
		}
		can, err := db.CanInsertImage(r.Context(), req.user)
		if err != nil {
			return
		}
		if !can {
			return errNoCandidatePost
		}

		// Limit data received to the maximum uploaded file size limit
		max := config.Get().MaxSize<<20 + 1<<10
		r.Body = http.MaxBytesReader(w, r.Body, int64(max))

		length, err := strconv.ParseUint(r.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 413,
			}
		}
		if uint(length) > max {
			return errTooLarge
		}
		err = r.ParseMultipartForm(0)
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 400,
			}
		}

		file, head, err := r.FormFile("image")
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 400,
			}
		}
		if uint(head.Size) > max {
			return errTooLarge
		}
		err = req.extract(r, head.Filename)
		if err != nil {
			return
		}

		select {
		case err = <-requestThumbnailing(thumbnailingRequest{
			insertionRequest: req,
			file:             file,
			size:             int(head.Size),
		}):
		case <-req.ctx.Done():
			return
		}
		if err == io.EOF {
			err = common.StatusError{
				Err:  err,
				Code: 400,
			}
		}
		return
	})
}

// Apply security restrictions to uploader
func validateUploader(w http.ResponseWriter, r *http.Request) (
	user auth.AuthKey,
	err error,
) {
	user, err = auth.ExtractAuthKey(r)
	if err != nil {
		return
	}
	need, err := db.NeedCaptcha(r.Context(), user)
	if err != nil {
		return
	}
	if need {
		err = common.StatusError{
			Err:  errors.New("captcha required"),
			Code: 403,
		}
		return
	}
	db.IncrementSpamScore(user, config.Get().ImageScore)
	return
}

// Extract and validate common request data from request
func (req *insertionRequest) extract(r *http.Request, name string) (err error) {
	req.Spoiler = r.FormValue("spoiler") == "true"
	req.Name = name
	errStr := func() string {
		if len(req.Name) > 200 {
			return "image name too long"
		}
		req.Name = strings.TrimSpace(req.Name)
		if i := strings.LastIndexByte(req.Name, '.'); i != -1 {
			req.Name = req.Name[:i]
			if strings.HasSuffix(req.Name, ".tar") {
				req.Name = req.Name[:len(req.Name)-4]
			}
		}
		if !utf8.ValidString(req.Name) {
			req.Name = strings.ToValidUTF8(req.Name, string(utf8.RuneError))
		}
		if len(req.Name) == 0 {
			return "no image name"
		}
		return ""
	}()
	if errStr != "" {
		return common.StatusError{
			Err:  errors.New(errStr),
			Code: 400,
		}
	}
	return nil
}

// UploadImageHash attempts to skip image upload, if the file has already
// been thumbnailed and is stored on the server. The client sends an SHA1 hash
// of the file it wants to upload. The server looks up, if such a file is
// thumbnailed. If yes, generates and sends a new image allocation token to
// the client.
func UploadImageHash(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, func() (err error) {
		var req struct {
			insertionRequest
			id common.SHA1Hash
		}
		req.ctx = r.Context()

		req.user, err = validateUploader(w, r)
		if err != nil {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
		err = r.ParseForm()
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 400,
			}
		}
		err = req.id.UnmarshalText([]byte(r.FormValue("id")))
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 400,
			}
		}
		err = req.extract(r, r.FormValue("name"))
		if err != nil {
			return
		}

		return tryInsertExisting(req.insertionRequest, req.id)
	})
}

// Try finding and inserting an already processed image into the post
func tryInsertExisting(req insertionRequest, id common.SHA1Hash,
) error {
	return db.InTransaction(req.ctx, func(tx pgx.Tx) (err error) {
		img, err := db.GetImage(req.ctx, tx, id)
		switch err {
		case nil:
			return insertImage(tx, req, img)
		case pgx.ErrNoRows:
			return errNotProcessed
		default:
			return
		}
	})
}

// Try inserting an image into the post
func insertImage(tx pgx.Tx, req insertionRequest, img common.ImageCommon,
) (err error) {
	var post, thread uint64
	post, thread, err = db.InsertImage(
		req.ctx,
		tx,
		req.user,
		img.SHA1,
		req.Name,
		req.Spoiler,
	)
	switch err {
	case nil:
		return websockets.InsertImage(thread, post, common.Image{
			ImageCommon: img,
			Spoilered:   req.Spoiler,
			Name:        req.Name,
		})
	case pgx.ErrNoRows:
		return errNoCandidatePost
	default:
		return
	}
}

// handleError sends the client file upload errors and logs them server-side
func handleError(w http.ResponseWriter, r *http.Request, f func() error) {
	err := f()
	if err == nil {
		return
	}

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
		ip = net.IPv4zero
	}
	log.Errorf("upload error: by %s: %s: %#v", ip, err, err)
}

// Create a new thumbnail, commit its resources to the DB and filesystem,
// insert it into an open post and send insertion even to listening clients
func insertNewThumbnail(req thumbnailingRequest, id common.SHA1Hash) (err error) {
	var img common.ImageCommon
	img.SHA1 = id

	conf := config.Get()
	thumb, err := processFile(req.file, &img, thumbnailer.Options{
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
	defer func() {
		if thumb != nil {
			putThumbBuffer(thumb)
		}
	}()
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
	err = db.InTransaction(req.ctx, func(tx pgx.Tx) (err error) {
		var thumbR io.ReadSeeker
		if thumb != nil {
			thumbR = bytes.NewReader(thumb)
		}
		err = db.AllocateImage(req.ctx, tx, img, req.file, thumbR)
		if err != nil {
			return
		}
		return insertImage(tx, req.insertionRequest, img)
	})
	return
}

// Separate function for easier testability
func processFile(
	f multipart.File,
	img *common.ImageCommon,
	opts thumbnailer.Options,
) (
	thumb []byte,
	err error,
) {
	src, thumbImage, err := thumbnailer.Process(f, opts)
	defer func() {
		// Add image internal buffer to pool.
		// Only image type used in thumbnailer by default.
		img, ok := thumbImage.(*image.RGBA)
		if ok {
			putThumbBuffer(img.Pix)
		}
	}()
	switch err {
	case nil:
		if config.Get().JPEGThumbnails {
			img.ThumbType = common.JPEG
		} else {
			img.ThumbType = common.WEBP
		}
	case thumbnailer.ErrCantThumbnail:
		err = nil
		img.ThumbType = common.NoFile
	default:
		return
	}

	img.FileType = mimeTypes[src.Mime]

	img.Audio = src.HasAudio
	img.Video = src.HasVideo
	img.Duration = uint32(src.Length / time.Second)

	// Some media has retardedly long meta strings. Just truncate them, instead
	// of rejecting.
	if len(src.Artist) > 100 {
		src.Artist = src.Artist[:100]
	}
	if len(src.Title) > 200 {
		src.Title = src.Title[:200]
	}
	if src.Artist != "" {
		img.Artist = &src.Artist
	}
	if src.Title != "" {
		img.Title = &src.Title
	}

	img.Width = uint16(src.Width)
	img.Height = uint16(src.Height)
	if thumbImage != nil {
		b := thumbImage.Bounds()
		img.ThumbWidth = uint16(b.Dx())
		img.ThumbHeight = uint16(b.Dy())
	}

	n, err := hashFile(img.MD5[:], f, md5.New())
	if err != nil {
		return
	}
	img.Size = uint64(n)

	if thumbImage != nil {
		w := bytes.NewBuffer(getThumbBuffer())
		switch img.ThumbType {
		case common.JPEG:
			err = jpeg.Encode(w, thumbImage, &jpeg.Options{
				Quality: 90,
			})
		case common.WEBP:
			err = webp.Encode(w, thumbImage, &webp.Options{
				Lossless: false,
				Quality:  90,
			})
		}
		if err != nil {
			return
		}
		thumb = w.Bytes()
	}

	return
}
