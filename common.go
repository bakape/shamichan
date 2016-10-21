// Methods common to both video and audio files

package goffmpeg

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include <libavformat/avformat.h>
import "C"
import "time"

// CodecName returns hte codec name of the best stream of type typ in the input
// or an empty string, if there is no stream of this type
func (c *Context) CodecName(typ MediaType) (string, error) {
	ci, err := c.codecContext(typ)
	if err == nil {
		return C.GoString(ci.ctx.codec.name), nil
	}
	fferr, ok := err.(FFmpegError)
	if ok && C.int(fferr) == C.AVERROR_STREAM_NOT_FOUND {
		return "", nil
	}
	return "", err
}

// Duration returns the duration of the input
func (c *Context) Duration() time.Duration {
	return time.Duration(c.avFormatCtx.duration * 1000)
}

// Bitrate returns the estimated bitrate in bits per second
func (c *Context) Bitrate() int64 {
	return int64(c.avFormatCtx.bit_rate)
}
