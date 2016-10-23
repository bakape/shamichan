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

func processOGG(data []byte) thumbResponse {
	return processMediaContainer(data, "theora", "vorbis", "opus")
}

func processMP4(data []byte) thumbResponse {
	return processMediaContainer(data, "h264", "mp3", "aac")
}

// Verify the media container file (OGG, MP4, etc.) contains the supported
// stream codecs and produce an appropriate thumbnail
func processMediaContainer(
	data []byte,
	videoC, audioC1, audioC2 string,
) (res thumbResponse) {
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
	case audioC1, audioC2:
		res.audio = true
	}

	video, err := c.CodecName(goffmpeg.Video)
	if err != nil {
		res.err = err
		return
	}
	res.video = video == videoC

	// Contains no streams compatible with the OGG container in common browsers
	if !res.audio && !res.video {
		res.err = errNoCompatibleStreams
		return
	}

	res.length = uint32(c.Duration() / 1000000000)

	if !res.video {
		// Can contain cover art
		if !c.HasImage() {
			return assignFallbackCover(res)
		}
		res.thumb, res.dims, res.err = processImage(c.Picture())
		return
	}

	return thumbnailVideo(c, res)
}
