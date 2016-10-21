// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"bytes"

	"github.com/bakape/goffmpeg"
)

// Extract data and thumbnail from a WebM video
func processWebm(data []byte) (res thumbResponse) {
	c, err := goffmpeg.NewContextReadSeeker(bytes.NewReader(data))
	if err != nil {
		res.err = err
		return
	}
	defer c.Close()

	audio, err := c.CodecName(goffmpeg.Audio)
	if err != nil {
		res.err = err
		return
	}
	if audio != "" {
		res.audio = true
	}

	res.length = uint32(c.Duration() / 1000000000)

	src, err := c.Thumbnail()
	if err != nil {
		res.err = err
		return
	}

	res.thumb, res.dims, res.err = verifyAndScale(src, "png")
	return
}
