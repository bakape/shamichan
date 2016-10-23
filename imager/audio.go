package imager

import (
	"bytes"
	"io/ioutil"
	"path/filepath"

	"github.com/bakape/goffmpeg"
)

// Directory for static image asset storage. Overridable for tests.
var assetRoot = "www"

// Fallback image for MP3 files with no cover
const fallbackCover = "audio-fallback.png"

// Test if file is an MP3
func detectMP3(buf []byte) (bool, error) {
	c, err := goffmpeg.NewContextReadSeeker(bytes.NewReader(buf))
	if err != nil {
		// Invalid file that can't even have a context created
		if fferr, ok := err.(goffmpeg.FFmpegError); ok && fferr.Code() == -1 {
			return false, nil
		}
		return false, err
	}
	defer c.Close()
	codec, err := c.CodecName(goffmpeg.Audio)
	if err != nil {
		return false, err
	}
	return codec == "mp3", nil
}

// Extract image and meta info from MP3 files and send them down the
// thumbnailing pipeline.
func processMP3(data []byte) (res thumbResponse) {
	c, err := goffmpeg.NewContextReadSeeker(bytes.NewReader(data))
	if err != nil {
		res.err = err
		return
	}
	defer c.Close()
	res.length = uint32(c.Duration() / 1000000000)

	// No cover art in file. Assign fallback cover and return.
	if !c.HasImage() {
		return assignFallbackCover(res)
	}

	res.thumb, res.dims, res.err = processImage(c.Picture())
	return
}

// Assign fallback cover art to audio file without any
func assignFallbackCover(res thumbResponse) thumbResponse {
	path := filepath.Join(assetRoot, fallbackCover)
	res.thumb, res.err = ioutil.ReadFile(path)
	res.dims = [4]uint16{150, 150, 150, 150}
	return res
}
