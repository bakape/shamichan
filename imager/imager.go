// Processes image files and images extracted from video, audio or PDF files

package imager

import (
	"errors"
	"image"
	jpegLib "image/jpeg"
	"io"

	"github.com/Soreil/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
)

var (
	errTooWide = errors.New("image too wide") // No such thing
	errTooTall = errors.New("image too tall")
)

// InitImager applies the thumbnail quality configuration
func InitImager() error {
	conf := config.Get().Images
	imager.JPEGOptions = jpegLib.Options{Quality: conf.JpegQuality}
	imager.PNGQuantization = conf.PngQuality
	return nil // To comply to the rest of the initialization functions
}

// Verify image parameters and create a thumbnail. The dims array contains
// [src_width, src_height, thumb_width, thumb_height].
func processImage(file io.ReadSeeker) ([]byte, [4]uint16, error) {
	file.Seek(0, 0)
	src, format, err := image.Decode(file)
	if err != nil {
		err = util.WrapError("error decoding source image", err)
		return nil, [4]uint16{}, err
	}

	dims, err := verifyDimentions(src)
	if err != nil {
		return nil, dims, err
	}

	scaled := imager.Scale(src, image.Point{X: 125, Y: 125})
	dims[2], dims[3] = getDims(scaled)
	thumbFormat := "png"
	if format == "jpeg" {
		thumbFormat = "jpeg"
	}
	thumb, err := imager.Encode(scaled, thumbFormat)
	if err != nil {
		return nil, dims, err
	}

	return thumb.Bytes(), dims, err
}

// Verify an image does not exceed the preset maximum dimentions and return them
func verifyDimentions(img image.Image) (dims [4]uint16, err error) {
	dims[0], dims[1] = getDims(img)
	conf := config.Get().Images.Max
	if dims[0] > conf.Width {
		err = errTooWide
		return
	}
	if dims[1] > conf.Height {
		err = errTooTall
	}
	return
}

// Calculates the width and height of an image
func getDims(img image.Image) (uint16, uint16) {
	rect := img.Bounds()
	return uint16(rect.Max.X - rect.Min.X), uint16(rect.Max.Y - rect.Min.Y)
}
