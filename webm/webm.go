package webm

import (
	"image"
	"io"
	"io/ioutil"

	"github.com/Soreil/video"
)

// 1A 45 DF A3 ....
const webmHeader = "???????????????????????????????webm"

func init() {
	image.RegisterFormat("webm", webmHeader, Decode, DecodeConfig)
}

//Decodes the first frame of a Webm video in to an image
func Decode(r io.Reader) (image.Image, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return video.Decode(b)
}

//Returns Webm metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return image.Config{}, err
	}
	return video.DecodeConfig(b)
}
