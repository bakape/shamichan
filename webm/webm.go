package webm

import (
	"image"
	"io"
	"io/ioutil"

	"github.com/Soreil/video"
)

//The integer field sizes can differ in a EBML (MKV) header.
func init() {
	image.RegisterFormat("webm", "\x1A\x45\xDF\xA3???????????????????????????webm", Decode, DecodeConfig)
	image.RegisterFormat("webm", "\x1A\x45\xDF\xA3????????????????????webm", Decode, DecodeConfig)
	image.RegisterFormat("webm", "\x1A\x45\xDF\xA3????????????webm", Decode, DecodeConfig)
	image.RegisterFormat("webm", "\x1A\x45\xDF\xA3????webm", Decode, DecodeConfig)
}

// Decode decodes the first frame of a Webm video in to an image
func Decode(r io.Reader) (image.Image, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return video.Decode(b)
}

// DecodeConfig returns Webm metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return image.Config{}, err
	}
	return video.DecodeConfig(b)
}
