// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"bytes"
	"errors"
	"image"

	"github.com/bakape/goffmpeg"
)

var (
	errNoCompatibleStreams = errors.New("no compatible streams found")
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

	return thumbnailVideo(c, res)
}

// Produce a thumbnail out of a video stream
func thumbnailVideo(c *goffmpeg.Context, res thumbResponse) thumbResponse {
	var src image.Image
	src, res.err = c.Thumbnail()
	if res.err != nil {
		return res
	}
	res.thumb, res.dims, res.err = verifyAndScale(src, "png")
	return res
}

// Verify and extract the contents of and OGG container and produce a thumbnail
func processOGG(data []byte) (res thumbResponse) {
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
	switch audio {
	case "vorbis", "opus":
		res.audio = true
	}

	video, err := c.CodecName(goffmpeg.Video)
	if err != nil {
		res.err = err
		return
	}
	res.video = video == "theora"

	// Contains no streams compatible with the OGG container in common browsers
	if !res.audio && !res.video {
		res.err = errNoCompatibleStreams
		return
	}

	res.length = uint32(c.Duration() / 1000000000)

	if !res.video {
		// OGG can contain cover art
		if !c.HasImage() {
			return assignFallbackCover(res)
		}
		res.thumb, res.dims, res.err = processImage(c.Picture())
		return
	}

	return thumbnailVideo(c, res)
}
