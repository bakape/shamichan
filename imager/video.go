// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"bytes"

	"github.com/bakape/video"
)

// Extract data and thumbnail from a WebM video
func processWebm(data []byte) (res thumbResponse) {
	d, err := video.NewDecoder(bytes.NewReader(data))
	if err != nil {
		res.err = err
		return
	}
	defer d.Close()

	audio, _, err := d.AVFormat()
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

	dur, err := d.Length()
	if err != nil {
		res.err = err
		return
	}
	res.length = uint32(dur / 1000000000)

	src, err := d.Thumbnail()
	if err != nil {
		res.err = err
		return
	}

	res.thumb, res.dims, res.err = verifyAndScale(src, "png")
	return
}
