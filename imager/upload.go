// Package imager handles image, video, etc. upload requests and processing
package imager

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/Soreil/apngdetector"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

var (
	// Map of net/http MIME types to the constants used internally
	mimeTypes = map[string]uint8{
		"image/jpeg":      types.JPEG,
		"image/png":       types.PNG,
		"image/gif":       types.GIF,
		"video/webm":      types.WEBM,
		"application/ogg": types.OGG,
		"video/mp4":       types.MP4,
		"application/zip": types.ZIP,
	}

	// File type tests for types not supported by http.DetectContentType
	typeTests = [...]struct {
		test  func([]byte) (bool, error)
		fType uint8
	}{
		{detectTarGZ, types.TGZ},
		{detectTarXZ, types.TXZ},
		{detect7z, types.SevenZip},
		{detectSVG, types.SVG},
		{detectMP3, types.MP3},
	}

	errTooLarge        = errors.New("file too large")
	errInvalidFileHash = errors.New("invalid file hash")

	isTest bool
)

// Response from a thumbnail generation performed concurently
type thumbResponse struct {
	audio, video bool
	dims         [4]uint16
	length       uint32
	thumb        []byte
	err          error
}

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(w http.ResponseWriter, r *http.Request) {
	// Limit data received to the maximum uploaded file size limit
	maxSize := config.Get().MaxSize * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	w.Header().Set("Access-Control-Allow-Origin", config.AllowedOrigin)

	code, id, err := newImageUpload(r)
	if err != nil {
		logError(w, r, code, err)
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
		logError(w, req, 500, err)
		return
	}
	hash := string(buf)

	_, err = db.FindImageThumb(hash)
	if err == r.ErrEmptyResult {
		w.Write([]byte("-1"))
		return
	} else if err != nil {
		logError(w, req, 500, err)
		return
	}

	code, token, err := db.NewImageToken(hash)
	if err != nil {
		logError(w, req, code, err)
	}
	w.Write([]byte(token))
}

// Send the client its error and log it
func logError(w http.ResponseWriter, r *http.Request, code int, err error) {
	text := err.Error()
	http.Error(w, text, code)
	if !isTest {
		log.Printf("upload error: %s: %s\n", auth.GetIP(r), text)
	}
}

// Separate function for cleaner error handling. Returns the HTTP status code of
// the response and error, if any.
func newImageUpload(req *http.Request) (int, string, error) {
	// Remove temporary files, when function returns
	defer func() {
		if req.MultipartForm != nil {
			if err := req.MultipartForm.RemoveAll(); err != nil {
				log.Printf("couldn't remove temporary files: %s\n", err)
			}
		}
	}()

	err := parseUploadForm(req)
	if err != nil {
		return 400, "", err
	}

	// TODO: A scheduler based on available RAM, so we don't run out of memory,
	// with concurrent burst loads.

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
	img, err := db.FindImageThumb(SHA1)
	noThumbnail := err == r.ErrEmptyResult
	if err != nil && !noThumbnail {
		return 500, "", err
	}

	// Already have a thumbnail
	if !noThumbnail {
		return db.NewImageToken(SHA1)
	}

	img.SHA1 = SHA1
	return newThumbnail(data, img)
}

// Parse and validate the form of the upload request
func parseUploadForm(req *http.Request) error {
	length, err := strconv.ParseInt(req.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}
	if length > config.Get().MaxSize*1024*1024 {
		return errTooLarge
	}
	return req.ParseMultipartForm(0)
}

// Create a new thumbnail, commit its resources to the DB and filesystem, and
// pass the image data to the client.
func newThumbnail(data []byte, img types.ImageCommon) (int, string, error) {
	fileType, err := detectFileType(data)
	if err != nil {
		return 400, "", err
	}

	// Generate MD5 hash and thumbnail concurently
	md5 := genMD5(data)
	thumb := processFile(data, fileType)

	if fileType == types.PNG {
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

	if err := db.AllocateImage(data, res.thumb, img); err != nil {
		return 500, "", err
	}

	return db.NewImageToken(img.SHA1)
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

// TODO: Waiting on Soreil
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
		case types.WEBM:
			res = processWebm(data)
		case types.MP3:
			res = processMP3(data)
		case types.OGG:
			res = processOGG(data)
		case types.MP4:
			res = processMP4(data)
		case types.ZIP, types.SevenZip, types.TGZ, types.TXZ:
			res = processArchive()
		case types.JPEG, types.PNG, types.GIF:
			res.thumb, res.dims, res.err = processImage(data)
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
		ch <- hex.EncodeToString(sum[:])
	}()
	return ch
}
