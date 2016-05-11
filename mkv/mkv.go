package mkv

import (
	"image"
	"io"
	"io/ioutil"

	"github.com/Soreil/video"
)

// 1A 45 DF A3
const mkvHeader = "???????????????????????????matroska"

func init() {
	image.RegisterFormat("mkv", mkvHeader, Decode, DecodeConfig)
}

//Decodes the first frame of an mkv video in to an image
func Decode(r io.Reader) (image.Image, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return video.Decode(b)
}

//Returns mkv metadata
func DecodeConfig(r io.Reader) (image.Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return image.Config{}, err
	}
	return video.DecodeConfig(b)
}
