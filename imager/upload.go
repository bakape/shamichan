// Package imager handles image, video, etc. upload requests and processing.
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
	"time"

	"github.com/Soreil/apngdetector"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/server/websockets"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
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

// Response from a thumbnail generation performed concurently
type thumbResponse struct {
	thumb []byte
	dims  [4]uint16
	err   error
}

// Spoiler ID is unparsable or not enabled
type errInvalidSpoiler string

func (e errInvalidSpoiler) Error() string {
	return "invalid spoiler ID: " + string(e)
}

// NewImageUpload  handles the clients' image (or other file) upload request
func NewImageUpload(res http.ResponseWriter, req *http.Request) {
	// Limit data received to the maximum uploaded file size limit
	conf := config.Get()
	req.Body = http.MaxBytesReader(res, req.Body, conf.MaxSize)
	res.Header().Set("Access-Control-Allow-Origin", conf.Origin)

	code, err := newImageUpload(req)
	if err != nil {
		text := err.Error()
		http.Error(res, text, code)
		log.Printf("upload error: %s: %s\n", req.RemoteAddr, text)
	}
}

// Separate function for cleaner error handling. Returns the HTTP status code of
// the response and error, if any.
func newImageUpload(req *http.Request) (int, error) {
	// Remove temporary files, when function returns
	defer func() {
		if req.MultipartForm != nil {
			if err := req.MultipartForm.RemoveAll(); err != nil {
				log.Printf("couldn't remove temporary files: %s\n", err)
			}
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

	sum := sha1.Sum(data)
	SHA1 := hex.EncodeToString(sum[:])
	img, err := FindImageThumb(SHA1)
	noThumbnail := err == r.ErrEmptyResult
	if err != nil && !noThumbnail {
		return 500, err
	}
	img.Imgnm = fileHeader.Filename
	img.Spoiler = spoiler

	// Already have a thumbnail
	if !noThumbnail {
		return passImage(img, client)
	}
	img.SHA1 = SHA1

	return newThumbnail(data, img, client)
}

// Parse and validate the form of the upload request
func parseUploadForm(req *http.Request) (
	clientID string, spoiler uint8, err error,
) {
	length, err := strconv.ParseInt(req.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return
	}
	if length > config.Get().MaxSize {
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
func extractSpoiler(req *http.Request) (uint8, error) {
	// Read the spoiler the client had chosen for the image, if any
	unparsed := req.FormValue("spoiler")
	if unparsed == "" {
		return 0, nil
	}

	unconverted, err := strconv.ParseUint(unparsed, 10, 8)
	if err != nil {
		return 0, errInvalidSpoiler(unparsed)
	}

	sp := uint8(unconverted)
	if !isValidSpoiler(sp) {
		return 0, errInvalidSpoiler(unparsed)
	}

	return sp, nil
}

// Confirms a spoiler exists in configuration
func isValidSpoiler(id uint8) bool {
	for _, valid := range config.Get().Spoilers {
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
		if err := DeallocateImage(img.SHA1); err != nil {
			log.Printf("counld't deallocate image: %s", img.SHA1)
		}
		return 408, errUsageTimeout
	}
}

// Create a new thumbnail, commit its resources to the DB and filesystem, and
// pass the image data to the client.
func newThumbnail(data []byte, img types.Image, client *websockets.Client) (
	int, error,
) {
	fileType, err := detectFileType(data)
	if err != nil {
		return 400, err
	}

	// Generate MD5 hash and thumbnail concurently
	md5 := genMD5(data)
	thumb := processFile(data, fileType)

	if fileType == png {
		img.APNG = apngdetector.Detect(data)
	}

	img.FileType = fileType
	img.Size = len(data)
	img.MD5 = <-md5
	res := <-thumb
	if res.err != nil {
		return 400, res.err
	}
	img.Dims = res.dims

	if err := allocateImage(data, res.thumb, img); err != nil {
		return 500, err
	}

	return passImage(img, client)
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
			return 0, fmt.Errorf("unsupported file type: %s", mimeType)
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

// Concurently delegate the processing of the file to an apropriate function by file
// type
func processFile(data []byte, fileType uint8) <-chan thumbResponse {
	ch := make(chan thumbResponse)

	go func() {
		var res thumbResponse
		switch fileType {

		// TODO: WebM thumbnailing
		// case webm:
		// 	return processWebm(file)

		case jpeg, png, gif:
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
