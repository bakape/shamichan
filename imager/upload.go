// Package imager handles image, video, etc. upload requests and processing.
package imager

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Soreil/apngdetector"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/server/websockets"
	"github.com/bakape/meguca/types"
)

// Supported file formats
const (
	jpeg = iota
	png
	gif
	webm
	pdf
	svg
	mp4
	mp3
	ogg
)

var (
	// Map of oficial MIME types to the extension representations we deal with
	mimeTypes = map[string]uint8{
		"image/jpeg":      jpeg,
		"image/png":       png,
		"image/gif":       gif,
		"video/webm":      webm,
		"application/pdf": pdf,
	}

	// Overridable for tests
	allocationTimeout = time.Second * 10

	errUsageTimeout = errors.New("image usage timeout")
)

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(res http.ResponseWriter, req *http.Request) {
	// Limit data received to the maximum uploaded file size limit
	conf := config.Get()
	req.Body = http.MaxBytesReader(res, req.Body, conf.Images.Max.Size)
	res.Header().Set("Access-Control-Allow-Origin", conf.HTTP.Origin)

	code, err := newImageUpload(req)
	if err != nil {
		text := err.Error()
		http.Error(res, text, code)
		log.Printf("Upload error: %s : %s\n", req.RemoteAddr, text)
	}
}

// Separate function for cleaner error handling. Returns the HTTP status code of
// the response and error, if any.
func newImageUpload(req *http.Request) (int, error) {
	// Remove temporary files, when function returns
	defer func() {
		if err := req.MultipartForm.RemoveAll(); err != nil {
			log.Printf("Error removing temporary files: %s\n", err)
		}
	}()

	clientID, spoiler, err := parseUploadForm(req)
	if err != nil {
		return 400, err
	}

	client, err := websockets.Clients.Get(clientID)
	if err != nil {
		return 400, err
	}

	file, fileHeader, err := req.FormFile("image")
	if err != nil {
		return 400, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return 500, err
	}

	sha1Sum := sha1.Sum(data)
	SHA1 := string(sha1Sum[:])
	img, err := FindImageThumb(SHA1)
	if err != nil {
		return 500, err
	}
	img.Imgnm = fileHeader.Filename
	img.Spoiler = spoiler

	// Already have a thumbnail
	if img.File != "" {
		return passImage(img, client)
	}

	fileType, err := detectFileType(data)
	if err != nil {
		return 400, err
	}
	img.FileType = fileType

	md5Sum := md5.Sum(data)
	img.MD5 = string(md5Sum[:])

	reader := bytes.NewReader(data)
	if fileType == png {
		img.APNG, err = apngdetector.Detect(reader)
		if err != nil {
			return 500, err
		}
		reader.Seek(0, 0)
	}

	mid, thumb, err := processFile(reader, img)
	if err != nil {
		return 400, err
	}
	reader.Seek(0, 0)
	if err := allocateImage(reader, thumb, mid, img); err != nil {
		return 500, err
	}

	return passImage(img, client)
}

// Parse and validate the form of the upload request
func parseUploadForm(req *http.Request) (
	clientID string, spoiler uint8, err error,
) {
	length, err := strconv.ParseInt(req.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return
	}
	if length > config.Get().Images.Max.Size {
		err = errors.New("File too large")
		return
	}

	err = req.ParseMultipartForm(0)
	if err != nil {
		return
	}

	clientID = req.FormValue("id")
	if clientID == "" {
		err = errors.New("No client ID specified")
		return
	}
	spoiler, err = extractSpoiler(req)

	return
}

// Extracts and validates a spoiler number from the form
func extractSpoiler(req *http.Request) (sp uint8, err error) {
	// Read the spoiler the client had chosen for the image, if any
	if unparsed := req.FormValue("spoiler"); unparsed != "" {
		var unconverted int
		unconverted, err = strconv.Atoi(unparsed)
		sp = uint8(unconverted)
		if !(err == nil && isValidSpoiler(sp)) {
			err = fmt.Errorf("Invalid spoiler ID: %s", unparsed)
		}
	}
	return
}

// Confirms a spoiler exists in configuration
func isValidSpoiler(id uint8) bool {
	for _, valid := range config.Get().Images.Spoilers {
		if id == valid {
			return true
		}
	}
	return false
}

// Passes the image struct to the requesting client. If the image is not
// succesfully passed in 10 seconds, it is deallocated.
func passImage(img types.Image, client *websockets.Client) (int, error) {
	select {
	case client.AllocateImage <- img:
		return 200, nil
	case <-time.Tick(allocationTimeout):
		if err := DeallocateImage(img.File); err != nil {
			log.Printf("counld't deallocate image: %s", img.File)
		}
		return 408, errUsageTimeout
	}
}

// detectFileType detects if the upload is of a supported file type, by reading
// its first 512 bytes. OGG and MP4 are also cheked to contain HTML5 supported
// video and audio streams.
func detectFileType(buf []byte) (uint8, error) {
	mimeType := http.DetectContentType(buf)
	mime, ok := mimeTypes[mimeType]
	if !ok {
		switch {
		case detectSVG(buf):
			return svg, nil
		case detectMP3(buf):
			return mp3, nil
		default:
			is, err := detectCompatibleMP4(buf)
			if is {
				return mp4, err
			}
			is, err = detectCompatibleOGG(buf)
			if is {
				return ogg, err
			}
			return 0, fmt.Errorf("Unsupported mime type: %s", mimeType)
		}
	}
	return mime, nil
}

// TODO: Waiting on Soreil

func detectSVG(buf []byte) bool {
	return false
}

func detectMP3(buf []byte) bool {
	return false
}

func detectCompatibleOGG(buf []byte) (bool, error) {
	return false, nil
}

func detectCompatibleMP4(buf []byte) (bool, error) {
	return false, nil
}

// Delegate the processing of the file to an apropriate function by file type
func processFile(file io.ReadSeeker, img types.Image) (
	io.Reader, io.Reader, error,
) {
	switch img.FileType {
	// case webm:
	// 	return processWebm(file)
	case jpeg, png, gif:
		return processImage(file)
	default:
		return nil, nil, fmt.Errorf("File type slipped in: %d", img.FileType)
	}
}
