// Validation and image extraction for webm and MP4/OGG with video

package imager

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include <libavutil/pixdesc.h>
// #include "video.h"
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"unsafe"
)

var (
	errNoCompatibleStreams = errors.New("no compatible streams found")
)

// Thumbnail extracts the first frame of the video
func (c *Context) Thumbnail() (image.Image, error) {
	ci, err := c.codecContext(Video)
	if err != nil {
		return nil, err
	}

	var f *C.AVFrame
	eErr := C.extract_video_image(&f, c.avFormatCtx, ci.ctx, ci.stream)
	if eErr != 0 {
		return nil, FFmpegError(eErr)
	}
	if f == nil {
		return nil, errors.New("failed to get frame")
	}
	defer C.av_frame_free(&f)

	if C.GoString(C.av_get_pix_fmt_name(int32(f.format))) != "yuv420p" {
		return nil, fmt.Errorf(
			"expected format: %s; got: %s",
			image.YCbCrSubsampleRatio420,
			C.GoString(C.av_get_pix_fmt_name(int32(f.format))),
		)
	}
	y := C.GoBytes(unsafe.Pointer(f.data[0]), f.linesize[0]*f.height)
	u := C.GoBytes(unsafe.Pointer(f.data[1]), f.linesize[0]*f.height/4)
	v := C.GoBytes(unsafe.Pointer(f.data[2]), f.linesize[0]*f.height/4)

	return &image.YCbCr{
		Y:              y,
		Cb:             u,
		Cr:             v,
		YStride:        int(f.linesize[0]),
		CStride:        int(f.linesize[0]) / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect: image.Rectangle{
			Min: image.Point{
				X: 0,
				Y: 0,
			},
			Max: image.Point{
				X: int(f.width),
				Y: int(f.height) / 2 * 2,
			},
		},
	}, nil
}

// Extract data and thumbnail from a WebM video
func processWebm(data []byte) (res thumbResponse) {
	c, err := NewContextReadSeeker(bytes.NewReader(data))
	if err != nil {
		res.err = err
		return
	}
	defer c.Close()

	audio, err := c.CodecName(Audio)
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
func thumbnailVideo(c *Context, res thumbResponse) thumbResponse {
	var src image.Image
	src, res.err = c.Thumbnail()
	if res.err != nil {
		return res
	}

	w := new(bytes.Buffer)
	res.err = png.Encode(w, src)
	if res.err != nil {
		return res
	}

	res.thumb, res.dims, res.err = processImage(w.Bytes())
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
	c, err := NewContextReadSeeker(bytes.NewReader(data))
	if err != nil {
		res.err = err
		return
	}
	defer c.Close()

	audio, err := c.CodecName(Audio)
	if err != nil {
		res.err = err
		return
	}
	switch audio {
	case audioC1, audioC2:
		res.audio = true
	}

	video, err := c.CodecName(Video)
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
