// Validation and image extraction for webm and MP4/OGG with video

package imager

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include "video.h"
import "C"
import (
	"errors"
	"unsafe"
)

var errNoCompatibleStreams = errors.New("no compatible streams found")

// Thumbnail extracts the first frame of the video
func (c *ffContext) Thumbnail() (
	buf []byte, width uint, height uint, err error,
) {
	ci, err := c.codecContext(ffVideo)
	if err != nil {
		return
	}

	var img C.struct_Buffer
	eErr := C.extract_video_image(&img, c.avFormatCtx, ci.ctx, ci.stream)
	switch {
	case eErr != 0:
		err = ffError(eErr)
		return
	case img.data == nil:
		err = errors.New("failed to get frame")
		return
	default:
		buf = C.GoBytes(unsafe.Pointer(img.data), C.int(img.size))
		C.free(unsafe.Pointer(img.data))
		return buf, uint(img.width), uint(img.height), nil
	}
}

// Extract data and thumbnail from a WebM video
func processWebm(data []byte) (res thumbResponse) {
	c, err := newFFContext(data)
	if err != nil {
		res.err = err
		return
	}
	defer c.Close()

	audio, err := c.CodecName(ffAudio)
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
func thumbnailVideo(c *ffContext, res thumbResponse) thumbResponse {
	src, width, height, err := c.Thumbnail()
	if err != nil {
		res.err = err
		return res
	}

	res.thumb, res.dims, res.PNGThumb, res.err = processImage(
		src,
		width,
		height,
	)
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
	c, err := newFFContext(data)
	if err != nil {
		res.err = err
		return
	}
	defer c.Close()

	audio, err := c.CodecName(ffAudio)
	if err != nil {
		res.err = err
		return
	}
	switch audio {
	case audioC1, audioC2:
		res.audio = true
	}

	video, err := c.CodecName(ffVideo)
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
		res.thumb, res.dims, res.PNGThumb, res.err = processImage(
			c.Picture(),
			0,
			0,
		)
		return
	}

	return thumbnailVideo(c, res)
}
