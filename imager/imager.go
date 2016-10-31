// Processes image files and images extracted from video, audio or PDF files

package imager

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"

	"github.com/bakape/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/util"
)

var (
	errTooWide = errors.New("image too wide") // No such thing
	errTooTall = errors.New("image too tall")
)

// InitImager applies the thumbnail quality configuration
func InitImager() error {
	if err := assets.CreateDirs(); err != nil {
		return err
	}

	conf := config.Get()
	imager.JPEGOptions = jpeg.Options{Quality: conf.JPEGQuality}
	imager.PNGQuantization = conf.PNGQuality

	return nil // To comply to the rest of the initialization functions
}

// Verify image parameters and create a thumbnail. The dims array contains
// [src_width, src_height, thumb_width, thumb_height].
func processImage(data []byte) ([]byte, [4]uint16, error) {
	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		err = util.WrapError("error decoding source image", err)
		return nil, [4]uint16{}, err
	}
	return verifyAndScale(src, format)
}

// Separate function, so it can be used in a more optimized video thumbnailer
func verifyAndScale(src image.Image, format string) ([]byte, [4]uint16, error) {
	dims, err := verifyDimensions(src)
	if err != nil {
		return nil, dims, err
	}

	scaled := imager.Scale(src, image.Point{X: 150, Y: 150})
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

// Verify an image does not exceed the preset maximum dimensions and return them
func verifyDimensions(img image.Image) (dims [4]uint16, err error) {
	dims[0], dims[1] = getDims(img)
	conf := config.Get()
	if dims[0] > conf.MaxWidth {
		err = errTooWide
		return
	}
	if dims[1] > conf.MaxHeight {
		err = errTooTall
	}
	return
}

// Calculates the width and height of an image
func getDims(img image.Image) (uint16, uint16) {
	rect := img.Bounds()
	return uint16(rect.Max.X - rect.Min.X), uint16(rect.Max.Y - rect.Min.Y)
}
