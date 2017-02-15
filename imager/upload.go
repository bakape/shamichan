// Package imager handles image, video, etc. upload requests and processing
package imager

import (
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/Soreil/apngdetector"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
)

var (
	// Map of net/http MIME types to the constants used internally
	mimeTypes = map[string]uint8{
		"image/jpeg":      common.JPEG,
		"image/png":       common.PNG,
		"image/gif":       common.GIF,
		"video/webm":      common.WEBM,
		"application/ogg": common.OGG,
		"video/mp4":       common.MP4,
		"application/zip": common.ZIP,
		"application/pdf": common.PDF,
	}

	// File type tests for types not supported by http.DetectContentType
	typeTests = [...]struct {
		test  func([]byte) (bool, error)
		fType uint8
	}{
		{detect7z, common.SevenZip},
		{detectTarGZ, common.TGZ},
		{detectTarXZ, common.TXZ},
		{detectSVG, common.SVG},
		{detectMP3, common.MP3},
	}

	errTooLarge        = errors.New("file too large")
	errInvalidFileHash = errors.New("invalid file hash")

	isTest bool
)

// Response from a thumbnail generation performed concurently
type thumbResponse struct {
	audio, video, PNGThumb bool
	dims                   [4]uint16
	length                 uint32
	thumb                  []byte
	err                    error
}

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(w http.ResponseWriter, r *http.Request) {
	// Limit data received to the maximum uploaded file size limit
	maxSize := config.Get().MaxSize * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))

	code, id, err := ParseUpload(r)
	if err != nil {
		LogError(w, r, code, err)
	}
	w.Write([]byte(id))
}

// UploadImageHash attempts to skip image upload, if the file has already
// been thumbnailed and is stored on the server. The client sends and SHA1 hash
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
		log.Printf("upload error: %s: %s\n", auth.GetIP(r), text)
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

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return 500, "", err
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
	if length > uint64(config.Get().MaxSize*1024*1024) {
		return errTooLarge
	}
	return req.ParseMultipartForm(0)
}

// Create a new thumbnail, commit its resources to the DB and filesystem, and
// pass the image data to the client.
func newThumbnail(data []byte, img common.ImageCommon) (int, string, error) {
	fileType, err := detectFileType(data)
	if err != nil {
		return 400, "", err
	}

	// Generate MD5 hash and thumbnail concurently
	md5 := genMD5(data)
	thumb := processFile(data, fileType)

	if fileType == common.PNG {
		img.APNG = apngdetector.Detect(data)
	}

	img.FileType = fileType
	img.Size = len(data)
	img.MD5 = <-md5
	res := <-thumb
	if res.err != nil {
		return 400, "", res.err // Some errors aren't actually 400, but most are
	}
	img.Dims = res.dims
	img.Length = res.length
	img.Audio = res.audio
	img.Video = res.video
	if res.PNGThumb {
		img.ThumbType = common.PNG
	}

	if err := db.AllocateImage(data, res.thumb, img); err != nil {
		return 500, "", err
	}

	return newImageToken(img.SHA1)
}

// detectFileType detects if the upload is of a supported file type, by reading
// its first 512 bytes. OGG and MP4 are also checked to contain HTML5 supported
// video and audio streams.
func detectFileType(buf []byte) (uint8, error) {
	mimeType := http.DetectContentType(buf)
	mime, ok := mimeTypes[mimeType]
	if !ok {
		for _, t := range typeTests {
			match, err := t.test(buf)
			if err != nil {
				return 0, err
			}
			if match {
				return t.fType, nil
			}
		}

		return 0, fmt.Errorf("unsupported file type: %s", mimeType)
	}
	return mime, nil
}

// TODO: SVG support
func detectSVG(buf []byte) (bool, error) {
	return false, nil
}

// Concurently delegate the processing of the file to an appropriate function by
// file type
func processFile(data []byte, fileType uint8) <-chan thumbResponse {
	ch := make(chan thumbResponse)

	go func() {
		var res thumbResponse
		switch fileType {
		case common.WEBM:
			res = processWebm(data)
		case common.MP3:
			res = processMP3(data)
		case common.OGG:
			res = processOGG(data)
		case common.MP4:
			res = processMP4(data)
		case common.ZIP, common.SevenZip, common.TGZ, common.TXZ:
			res = processArchive()
		case common.JPEG, common.PNG, common.GIF, common.PDF:
			res.thumb, res.dims, res.PNGThumb, res.err = processImage(
				data,
				0,
				0,
			)
		}

		ch <- res
	}()

	return ch
}

// Concurently generates the MD5 hash of an image
func genMD5(data []byte) <-chan string {
	ch := make(chan string)
	go func() {
		sum := md5.Sum(data)
		ch <- base64.RawURLEncoding.EncodeToString(sum[:])
	}()
	return ch
}
