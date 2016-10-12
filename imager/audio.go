package imager

import (
	"bytes"
	"io/ioutil"
	"path/filepath"

	"github.com/Soreil/audio"
)

// Directory for static image asset storage. Overridable for tests.
var assetRoot = "www"

// Fallback image for MP3 files with no cover
const fallbackCover = "audio-fallback.png"

// Test if file is an MP3
func detectMP3(buf []byte) (bool, error) {
	d, err := audio.NewDecoder(bytes.NewReader(buf))
	if err != nil {
		if err == audio.DecoderError {
			return false, nil
		}
		return false, err
	}
	defer d.Destroy()
	return d.AudioFormat() == "mp3", nil
}

// Extract image and meta info from MP3 files and send them down the
// thumbnailing pipeline.
func processMP3(data []byte) (res thumbResponse) {
	d, err := audio.NewDecoder(bytes.NewReader(data))
	if err != nil {
		res.err = err
		return
	}
	defer d.Destroy()
	res.length = uint32(d.Duration() / 1000000000)

	// No cover art in file. Assign fallback cover and return.
	if !d.HasImage() {
		path := filepath.Join(assetRoot, fallbackCover)
		res.thumb, res.err = ioutil.ReadFile(path)
		res.dims = [4]uint16{150, 150, 150, 150}
		return
	}

	res.thumb, res.dims, res.err = processImage(d.Picture())
	return
}
