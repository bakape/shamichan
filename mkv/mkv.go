package mkv

import (
	"bytes"
	"image"
	"io"
	"io/ioutil"

	"github.com/bakape/video"
)

//The integer field sizes can differ in a EBML (MKV) header.
func init() {
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3???????????????????????????matroska", Decode, DecodeConfig)
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3????????????????????matroska", Decode, DecodeConfig)
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3????????????matroska", Decode, DecodeConfig)
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3????matroska", Decode, DecodeConfig)
}

// Decode decodes the first frame of an mkv video in to an image
func Decode(r io.Reader) (image.Image, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return video.Decode(bytes.NewReader(b))
}

// DecodeConfig returns mkv metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return image.Config{}, err
	}
	return video.DecodeConfig(bytes.NewReader(b))
}
