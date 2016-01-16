package server

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
)

// ProtoImage stores data of an image that is being processed as well as data,
// that will be stored, once the image finishes processing.
type ProtoImage struct {
	Image
	ClientID string
}

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(res http.ResponseWriter, req *http.Request) {
	// Limit data received to the maximum uploaded file size limit
	req.Body = http.MaxBytesReader(res, req.Body, config.Images.Max.Size)
	clientID, spoiler, err := parseUploadForm(req)
	if err != nil {
		passError(res, req, err, 400)
	}
	image := &ProtoImage{
		Image: Image{
			Spoiler: spoiler,
		},
		ClientID: clientID,
	}
	fmt.Printf("%#v\n", image)
}

// Pass error message to client and log server-side
func passError(
	res http.ResponseWriter,
	req *http.Request,
	err error,
	code int,
) {
	text := err.Error()
	http.Error(res, text, code)
	log.Printf("Upload error: %s : %s", req.RemoteAddr, text)
}

func parseUploadForm(req *http.Request) (id string, spoiler uint16, err error) {
	err = req.ParseMultipartForm(1073741824) // 10 MB
	if err != nil {
		return
	}
	id = req.FormValue("id")
	if id == "" {
		err = errors.New("Invalid client ID")
		return
	}

	// Read the spoiler the client had chosen for the image, if any
	if unparsed := req.FormValue("spoiler"); unparsed != "" {
		var unconverted int
		unconverted, err = strconv.Atoi(unparsed)
		spoiler = uint16(unconverted)
		if err != nil || !isValidSpoiler(spoiler) {
			err = errors.New("Invalid spoiler ID")
		}
	}
	return
}

func isValidSpoiler(id uint16) bool {
	for _, valid := range config.Images.Spoilers {
		if id == valid {
			return true
		}
	}
	return false
}

// Map of oficial MIME types to the extension representations we deal with
var mimeTypes = map[string]string{
	"image/jpeg":               ".jpg",
	"image/png":                ".png",
	"image/gif":                ".gif",
	"video/webm":               ".webm",
	"text/xml; charset=utf-8":  ".xml",
	"application/pdf":          ".pdf",
	"application/octet-stream": "unknown",
}

func detectFileType(req *http.Request, file multipart.File) error {
	first512 := make([]byte, 512)
	if _, err := file.Read(first512); err != nil {
		return err
	}
	mimeType := http.DetectContentType(first512)
	ext, ok := mimeTypes[mimeType]
	if ok {
		fmt.Println(ext)
	}
	return nil
}
