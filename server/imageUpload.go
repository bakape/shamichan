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
	iu.parseForm()

	file, _, err := iu.req.FormFile("image")
	if err != nil {
		iu.Error(400, "invalid", err)
		return
	}
	defer file.Close()
	iu.detectFileType(file)
}

func (iu *ImageUpload) parseForm() {
	if err := iu.req.ParseMultipartForm(1073741824); err != nil { // 10 MB
		iu.Error(500, "req_problem", err)
		return
	}
	if iu.clientID = iu.req.FormValue("id"); iu.clientID == "" {
		iu.Error(400, "bad_client", errors.New("Bad client ID"))
		return
	}

	// Read the spoiler the client had chosen for the image, if any
	if spoiler := iu.req.FormValue("spoiler"); spoiler != "" {
		spoilerID, err := strconv.ParseUint(spoiler, 10, 16)
		if err != nil {
			iu.Error(400, "invalid", err)
			return
		}
		iu.spoiler = uint16(spoilerID)
	}
}

// Writes the apropriate error status code and error message to the client
// and logs server-side.
func (iu *ImageUpload) Error(status int, code string, err error) {
	http.Error(iu.res, iu.lang[code], status)
	logError(iu.req, err)
}

func (iu *ImageUpload) detectFileType(file multipart.File) {
	first512 := make([]byte, 512)
	if _, err := file.Read(first512); err != nil {
		iu.Error(400, "req_problem", err)
		return
	}
	mimeType := http.DetectContentType(first512)
	ext, ok := mimeTypes[mimeType]
	if ok {
		fmt.Println(ext)
	}
}
