package mp4

import (
	"bytes"
	"image"
	"io"
	"io/ioutil"

	"github.com/bakape/video"
)

const mp4Header = "????ftyp"

func init() {
	image.RegisterFormat("mp4", mp4Header, Decode, DecodeConfig)
}

// Decode decodes the first frame of an mp4 video in to an image
func Decode(r io.Reader) (image.Image, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return video.Decode(bytes.NewReader(b))
}

// DecodeConfig returns mp4 metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return image.Config{}, err
	}
	return video.DecodeConfig(bytes.NewReader(b))
}
