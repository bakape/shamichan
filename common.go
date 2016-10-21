// Methods common to both video and audio files

package goffmpeg

// #cgo CFLAGS: -std=c11
import "C"
import "time"

// CodecName returns hte codec name of the best stream of type typ in the input
// or an empty string, if there is no stream of this type
func (c *Context) CodecName(typ MediaType) (string, error) {
	ci, err := c.codecContext(typ)
	if err == nil {
		return C.GoString(ci.ctx.codec.name), nil
	}
	if fferr, ok := err.(FFmpegError); ok && fferr.Code() == -1381258232 {
		return "", nil
	}
	return "", err
}

// Duration returns the duration of the input
func (c *Context) Duration() (time.Duration, error) {
	return time.Duration(c.avFormatCtx.duration * 1000), nil
}

// Bitrate returns the estimated bitrate in bits per second
func (c *Context) Bitrate() int64 {
	return int64(c.avFormatCtx.bit_rate)
}
