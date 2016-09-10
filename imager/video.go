// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"bytes"

	"github.com/Soreil/video"
)

// Extract data and thumbnail from a WebM video
func processWebm(data []byte) (res thumbResponse) {
	audio, _, err := video.DecodeAVFormat(data)
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

	dur, err := video.DecodeLength(bytes.NewReader(data))
	if err != nil {
		res.err = err
		return
	}
	res.length = int32(dur / 1000000000)

	res.thumb, res.dims, res.err = processImage(data)
	return
}
