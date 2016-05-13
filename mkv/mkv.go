package mkv

import (
	"bytes"
	"image"
	"io"
	"io/ioutil"
	"log"

	"github.com/Soreil/video"
)

//The integer field sizes can differ in a EBML (MKV) header.
func init() {
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3???????????????????????????matroska", Decode, DecodeConfig)
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3????????????????????matroska", Decode, DecodeConfig)
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3????????????matroska", Decode, DecodeConfig)
	image.RegisterFormat("mkv", "\x1A\x45\xDF\xA3????matroska", Decode, DecodeConfig)
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
	log.Println("Trash prefix length:", bytes.Index(b, []byte("matroska")))
	if err != nil {
		return image.Config{}, err
	}
	return video.DecodeConfig(b)
}
