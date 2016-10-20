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
	var f *C.AVFrame
	if err := C.extract_video_image(&f, c.avFormatCtx); err != 0 {
		return nil, FormatError(int(err))
	}
	if f == nil {
		return nil, errors.New("failed to get AVCodecContext")
	}

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
	cc, err := c.CodecContext(Video)
	if err != nil {
		return image.Config{}, err
	}
	if cc == nil {
		return image.Config{}, errors.New("failed to decode")
	}
	codecCtx := (*C.AVCodecContext)(cc)
	defer C.avcodec_free_context(&codecCtx)

	s := C.GoString(C.av_get_pix_fmt_name(int32(codecCtx.pix_fmt)))
	if strings.Contains(s, "yuv") {
		return image.Config{
			ColorModel: color.YCbCrModel,
			Width:      int(codecCtx.width),
			Height:     int(codecCtx.height),
		}, nil
	}

	return image.Config{
		ColorModel: color.RGBAModel,
		Width:      int(codecCtx.width),
		Height:     int(codecCtx.height),
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

// AVFormat returns contained stream codecs with desired codec name verbosity
func (c *Context) AVFormat(detailed bool) (audio, video string, err error) {
	video, err = c.CodecName(Video, detailed)
	if err != nil {
		return
	}
	audio, err = c.CodecName(Audio, detailed)
	return
}
