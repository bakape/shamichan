// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"bytes"
	"io"

	"github.com/bakape/video"
	// webm thumbnailing driver
	_ "github.com/bakape/video/webm"
)

// Extract data and thumbnail from a WebM video
func processWebm(data []byte) (res thumbResponse) {
	r := bytes.NewReader(data)
	audio, _, err := video.DecodeAVFormat(r)
	if err != nil {
		if err.Error() == "Failed to decode audio stream" {
			err = nil
		} else {
			res.err = err
			return
		}
	}
	if audio != "" {
		res.audio = true
	}

	r.Seek(0, io.SeekStart)
	dur, err := video.DecodeLength(r)
	if err != nil {
		res.err = err
		return
	}
	res.length = uint32(dur / 1000000000)

	res.thumb, res.dims, res.err = processImage(data)
	return
}
