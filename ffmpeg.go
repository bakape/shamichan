// Package video provides thumbnailing and meta information retrieval for video
// files
package video

// #cgo pkg-config: libavcodec libavutil libavformat libswscale
// #cgo CFLAGS: -std=c11
// #include <libavutil/pixdesc.h>
// #include "ffmpeg.h"
import "C"
import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"
	"time"
	"unsafe"

	"github.com/bakape/video/avio"
)

func init() {
	C.av_register_all()
	C.avcodec_register_all()
}

// Decode uses CGo FFmpeg binding to extract the first video frame
func Decode(r io.ReadSeeker) (image.Image, error) {
	ctx, err := avio.NewContext(&avio.Handlers{
		ReadPacket: r.Read,
		Seek:       r.Seek,
	})
	if err != nil {
		return nil, err
	}
	defer ctx.Free()

	f := C.extract_video_image(ctx.AVFormatContext())
	if f == nil {
		return nil, errors.New("Failed to Get AVCodecContext")
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

// DecodeConfig uses CGo FFmpeg binding to find video config
func DecodeConfig(r io.ReadSeeker) (image.Config, error) {
	ctx, err := avio.NewContext(&avio.Handlers{
		ReadPacket: r.Read,
		Seek:       r.Seek,
	})
	if err != nil {
		return image.Config{}, err
	}
	defer ctx.Free()

	f := C.extract_video(ctx.AVFormatContext())
	if f == nil {
		return image.Config{}, errors.New("Failed to decode")
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

// DecodeAVFormatDetail retrieves contained stream codecs in more verbose
// representation
func DecodeAVFormatDetail(r io.ReadSeeker) (audio, video string, err error) {
	ctx, err := avio.NewContext(&avio.Handlers{
		ReadPacket: r.Read,
		Seek:       r.Seek,
	})
	if err != nil {
		return
	}
	defer ctx.Free()

	avfc := ctx.AVFormatContext()
	f := C.extract_video(avfc)
	if f == nil {
		err = errors.New("Failed to decode video stream")
		return
	}
	video = C.GoString(f.codec.long_name)

	f = C.extract_audio(avfc)
	if f == nil {
		err = errors.New("Failed to decode audio stream")
		return
	}
	audio = C.GoString(f.codec.long_name)
	return
}

// DecodeAVFormat retrieves contained stream codecs
func DecodeAVFormat(r io.ReadSeeker) (audio, video string, err error) {
	ctx, err := avio.NewContext(&avio.Handlers{
		ReadPacket: r.Read,
		Seek:       r.Seek,
	})
	if err != nil {
		return
	}
	defer ctx.Free()

	avfc := ctx.AVFormatContext()
	f := C.extract_video(avfc)
	if f == nil {
		err = errors.New("Failed to decode video stream")
		return
	}
	video = C.GoString(f.codec.name)

	f = C.extract_audio(avfc)
	if f == nil {
		err = errors.New("Failed to decode audio stream")
		return
	}
	audio = C.GoString(f.codec.name)
	return
}

// DecodeLength returns the length of the video
func DecodeLength(r io.ReadSeeker) (time.Duration, error) {
	ctx, err := avio.NewContext(&avio.Handlers{
		ReadPacket: r.Read,
		Seek:       r.Seek,
	})
	if err != nil {
		return 0, err
	}
	defer ctx.Free()

	dur := (*C.struct_AVFormatContext)(ctx.AVFormatContext()).duration
	return time.Duration(dur * 1000), nil
}
