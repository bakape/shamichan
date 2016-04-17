// Processes image files and images extracted from video, audio or PDF files

package imager

import (
	"errors"
	"fmt"
	_ "github.com/Soreil/imager" // TEMP
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"image"
	"io"
	"mime/multipart"
)

// Verify image parameters and create thumbnails
func processImage(
	file multipart.File,
	fileHeader *multipart.FileHeader,
	img *ProtoImage,
) (string, error) {
	if err := verifyImage(file); err != nil {
		return "", err
	}
	return "", nil
}

// Verify image dimentions and that it has not been posted before in the
// configured time
func verifyImage(file io.Reader) error {
	stats, format, err := image.DecodeConfig(file)
	if err != nil {
		return util.WrapError("Error decoding image", err)
	}
	switch format {
	case "jpeg", "png", "gif":
	default:
		return fmt.Errorf("Unsupported image format: %s", format)
	}
	if err := verifyDimentions(stats); err != nil {
		return err
	}

	return nil
}

// Verify an image does not exceed the preset maximum dimentions
func verifyDimentions(stats image.Config) error {
	conf := config.Images().Max
	width := stats.Width
	height := stats.Height
	var err error
	switch {
	case width > conf.Width:
		err = errors.New("Image too wide")
	case height > conf.Height:
		err = errors.New("Image too tall")
	}
	return err
}
