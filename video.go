package goffmpeg

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include <libavutil/pixdesc.h>
// #include "video.h"
import "C"
import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"
	"unsafe"
)

// Register all supported formats with the image package
func init() {
	magic := [...]struct {
		format, seq string
	}{
		{"mkv", "\x1A\x45\xDF\xA3???????????????????????????matroska"},
		{"mkv", "\x1A\x45\xDF\xA3????????????????????matroska"},
		{"mkv", "\x1A\x45\xDF\xA3????????????matroska"},
		{"mkv", "\x1A\x45\xDF\xA3????matroska"},
		{"mp4", "????ftyp"},
		{"webm", "\x1A\x45\xDF\xA3???????????????????????????webm"},
		{"webm", "\x1A\x45\xDF\xA3????????????????????webm"},
		{"webm", "\x1A\x45\xDF\xA3????????????webm"},
		{"webm", "\x1A\x45\xDF\xA3????webm"},
	}
	for _, m := range magic {
		image.RegisterFormat(m.format, m.seq, Decode, DecodeConfig)
	}
}

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

// Decode extracts the first frame of the video
func Decode(r io.Reader) (image.Image, error) {
	d, err := NewContextReader(r)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.Thumbnail()
}

// Config uses CGo FFmpeg binding to find video's image.Config metadata
func (c *Context) Config() (image.Config, error) {
	ci, err := c.codecContext(Video)
	if err != nil {
		return image.Config{}, err
	}

	ctx := ci.ctx

	s := C.GoString(C.av_get_pix_fmt_name(int32(ctx.pix_fmt)))
	if strings.Contains(s, "yuv") {
		return image.Config{
			ColorModel: color.YCbCrModel,
			Width:      int(ctx.width),
			Height:     int(ctx.height),
		}, nil
	}

	return image.Config{
		ColorModel: color.RGBAModel,
		Width:      int(ctx.width),
		Height:     int(ctx.height),
	}, nil
}

// DecodeConfig uses CGo FFmpeg binding to find the video's image.Config
// metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	d, err := NewContextReader(r)
	if err != nil {
		return image.Config{}, err
	}
	defer d.Close()
	return d.Config()
}
