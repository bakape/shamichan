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

func parseUploadForm(req *http.Request) (id string, spoiler uint8, err error) {
	err = req.ParseMultipartForm(512)
	if err != nil {
		return
	}
	id = req.FormValue("id")
	if id == "" {
		err = errors.New("No client ID specified")
		return
	}

	// Read the spoiler the client had chosen for the image, if any
	if unparsed := req.FormValue("spoiler"); unparsed != "" {
		var unconverted int
		unconverted, err = strconv.Atoi(unparsed)
		if err != nil || !isValidSpoiler(spoiler) {
			err = fmt.Errorf("Invalid spoiler ID: %s", unparsed)
		}
		spoiler = uint8(unconverted)
	}
	return
}

func isValidSpoiler(id uint8) bool {
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
	"application/pdf":          ".pdf",
	"application/octet-stream": "unknown",
}

func detectFileType(file multipart.File) (string, error) {
	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return "", err
	}
	mimeType := http.DetectContentType(buf)
	if ext, ok := mimeTypes[mimeType]; ok {
		switch ext {
		case "unknown":
			switch {
			case detectSVG(buf):
				return ".svg", nil
			case detectMP3(buf):
				return ".mp3", nil
			}
		default:
			return ext, nil
		}
	}
	return "", fmt.Errorf("Unsupported mime type: %s", mimeType)
}

func detectSVG(buf []byte) bool {
	return false
}

func detectMP3(buf []byte) bool {
	return false
}
