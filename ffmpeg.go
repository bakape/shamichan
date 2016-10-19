// Package video provides thumbnailing and meta information retrieval for video
// files
package video

// #cgo pkg-config: libavcodec libavutil libavformat
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
	f, err := C.extract_video_image(d.avFormatCtx)
	if err != nil {
		return nil, err
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
	d, err := NewDecoderReader(r)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.Thumbnail()
}

// Config uses CGo FFmpeg binding to find video's image.Config metadata
func (d *Decoder) Config() (image.Config, error) {
	cc, err := d.CodecContext(avio.Video)
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

// DecodeConfig uses CGo FFmpeg binding to find video's image.Config metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	d, err := NewDecoderReader(r)
	if err != nil {
		return image.Config{}, err
	}
	defer d.Close()
	return d.Config()
}

// AVFormat returns contained stream codecs with desired codec name verbosity
func (d *Decoder) AVFormat(detailed bool) (audio, video string, err error) {
	video, err = d.CodecName(avio.Video, detailed)
	if err != nil {
		return
	}
	audio, err = d.CodecName(avio.Audio, detailed)
	return
}

// Length returns the length of the video
func (d *Decoder) Length() (time.Duration, error) {
	return time.Duration(d.avFormatCtx.duration * 1000), nil
}
