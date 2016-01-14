package server

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
)

// ImageUpload handles client-uploaded images, etc., verifies them, then
// generates thumbnails and stores image data in RethinkDB.
type ImageUpload struct {
	res      http.ResponseWriter
	req      *http.Request
	lang     map[string]string
	spoiler  uint16
	clientID string
}

// NewImageUpload creates a new ImageUpload instance, that handles the client's
// image (or other file) upload request.
func NewImageUpload(res http.ResponseWriter, req *http.Request) {
	iu := ImageUpload{
		res:  res,
		req:  req,
		lang: langs[chooseLang(req)].Imager,
	}
	iu.process()
}

var mimeTypes = map[string]string{
	"image/jpeg":               ".jpg",
	"image/png":                ".png",
	"image/gif":                ".gif",
	"video/webm":               ".webm",
	"text/xml; charset=utf-8":  ".xml",
	"application/pdf":          ".pdf",
	"application/octet-stream": "unknown",
}

// Main method, that starts the upload processing chain
func (iu *ImageUpload) process() {
	// Limit data received to the maximum uploaded file size limit
	iu.req.Body = http.MaxBytesReader(
		iu.res,
		iu.req.Body,
		config.Images.Max.Size,
	)
	if iu.req.Method != "POST" {
		iu.res.WriteHeader(405)
		return
	}
	iu.parseForm()

	file, _, err := iu.req.FormFile("image")
	if err != nil {
		iu.writeError(400, "invalid", err)
	}
	defer file.Close()
	iu.detectFileType(file)
}

func (iu *ImageUpload) parseForm() {
	if err := iu.req.ParseMultipartForm(1073741824); err != nil { // 10 MB
		iu.writeError(500, "req_problem", err)
	}
	if iu.clientID = iu.req.FormValue("id"); iu.clientID == "" {
		iu.writeError(400, "bad_client", errors.New("Bad client ID"))
	}

	// Read the spoiler the client had chosen for the image, if any
	if spoiler := iu.req.FormValue("spoiler"); spoiler != "" {
		spoilerID, err := strconv.ParseUint(spoiler, 10, 16)
		if err != nil {
			iu.writeError(400, "invalid", err)
		}
		iu.spoiler = uint16(spoilerID)
	}
}

type uploadError struct {
	ip    string
	inner error
}

func (e uploadError) Error() string {
	return fmt.Sprintf("Upload error by IP %v : %v", e.ip, e.inner.Error())
}

// Writes the apropriate error status code and error message to the client.
// Also panics, which terminates to goroutine and is logged server-side.
func (iu *ImageUpload) writeError(status int, code string, err error) {
	iu.res.WriteHeader(status)
	iu.res.Write([]byte(iu.lang[code]))
	panic(uploadError{iu.req.RemoteAddr, err})
}

func (iu *ImageUpload) detectFileType(file multipart.File) {
	first512 := make([]byte, 512)
	if _, err := file.Read(first512); err != nil {
		iu.writeError(400, "req_problem", err)
	}
	mimeType := http.DetectContentType(first512)
	ext, ok := mimeTypes[mimeType]
	if ok {
		fmt.Println(ext)
	}
}
