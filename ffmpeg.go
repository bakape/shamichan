// Package video provides thumbnailing and meta information retrieval for video
// files
package video

// #cgo pkg-config: libavcodec libavutil libavformat libswscale
// #cgo CFLAGS: -std=c11
// #include <libavutil/pixdesc.h>
// #include "ffmpeg.h"
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"strings"
	"time"
	"unsafe"

	"github.com/bakape/video/avio"
)

// Decoder wraps around internal state, all methods are called on this.
type Decoder struct {
	avio.Context
	avIOCtx     *C.struct_AVIOContext
	avFormatCtx *C.struct_AVFormatContext
}

func init() {
	C.av_register_all()
	C.avcodec_register_all()

	// Register all supported formats with the image package
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

// NewDecoder sets up a context for the file. Call methods on it to perform
// operations on the file.
func NewDecoder(r io.ReadSeeker) (*Decoder, error) {
	ctx, err := avio.NewContext(&avio.Handlers{
		ReadPacket: r.Read,
		Seek:       r.Seek,
	})
	if err != nil {
		return nil, err
	}

	return &Decoder{
		Context: *ctx,
		// C types from different packages are not equal. Cast them.
		avIOCtx:     (*C.struct_AVIOContext)(ctx.AVIOContext()),
		avFormatCtx: (*C.struct_AVFormatContext)(ctx.AVFormatContext()),
	}, nil
}

// NewDecoderReader reads the entirety of r and returns a Decoder to operate on
// the contents
func NewDecoderReader(r io.Reader) (*Decoder, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewDecoder(bytes.NewReader(b))
}

// Thumbnail extracts the first frame of the video
func (d *Decoder) Thumbnail() (image.Image, error) {
	f := C.extract_video_image(d.avFormatCtx)
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
	d, err := NewDecoderReader(r)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.Thumbnail()
}

// Config uses CGo FFmpeg binding to find video's image.Config metadata
func (d *Decoder) Config() (image.Config, error) {
	f := C.extract_video(d.avFormatCtx)
	if f == nil {
		return image.Config{}, errors.New("failed to decode")
	}

	s := C.GoString(C.av_get_pix_fmt_name(int32(f.pix_fmt)))
	if strings.Contains(s, "yuv") {
		return image.Config{
			ColorModel: color.YCbCrModel,
			Width:      int(f.width),
			Height:     int(f.height),
		}, nil
	}

	return image.Config{
		ColorModel: color.RGBAModel,
		Width:      int(f.width),
		Height:     int(f.height),
	}, nil
}

// DecodeConfig uses CGo FFmpeg binding to find video's image.Config metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	d, err := NewDecoderReader(r)
	if err != nil {
		return image.Config{}, err
	}
	defer d.Close()
	return d.Config()
}

// AVFormatDetail returns contained stream codecs in a more verbose
// representation
func (d *Decoder) AVFormatDetail() (audio, video string, err error) {
	f := C.extract_video(d.avFormatCtx)
	if f == nil {
		err = errors.New("failed to decode video stream")
		return
	}
	video = C.GoString(f.codec.long_name)

	f = C.extract_audio(d.avFormatCtx)
	if f == nil {
		err = errors.New("failed to decode audio stream")
		return
	}
	audio = C.GoString(f.codec.long_name)
	return
}

// AVFormat returns contained stream codecs
func (d *Decoder) AVFormat() (audio, video string, err error) {
	f := C.extract_video(d.avFormatCtx)
	if f == nil {
		err = errors.New("Failed to decode video stream")
		return
	}
	video = C.GoString(f.codec.name)

	f = C.extract_audio(d.avFormatCtx)
	if f == nil {
		err = errors.New("Failed to decode audio stream")
		return
	}
	audio = C.GoString(f.codec.name)
	return
}

// Length returns the length of the video
func (d *Decoder) Length() (time.Duration, error) {
	return time.Duration(d.avFormatCtx.duration * 1000), nil
}
