// Processes image files and images extracted from video, audio or PDF files

package imager

import (
	"errors"
	"fmt"
	"image"
	"io"

	"github.com/Soreil/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
)

var (
	// Maximum dimentions for a normal small thumbnail
	normal = image.Point{X: 125, Y: 125}

	// Maximum dimentions for a high quality thumbnail
	sharp = image.Point{X: 250, Y: 250}
)

// Verify image parameters and create thumbnails
func processImage(file io.ReadSeeker) (
	large io.Reader, small io.Reader, err error,
) {
	err = verifyImage(file)
	if err != nil {
		return
	}

	// if img.FileType == png {
	// 	img.APNG = apngdetector.Detect(data)
	// }

	file.Seek(0, 0)
	thumbs, _, err := imager.Thumbnails(file, sharp, normal)
	if err != nil {
		return
	}

	large = thumbs[0]
	small = thumbs[1]
	return
}

// Verify image dimentions and that it has not been posted before in the
// configured time
func verifyImage(file io.ReadSeeker) error {
	decoded, format, err := image.Decode(file)
	if err != nil {
		return util.WrapError("Error decoding image", err)
	}

	switch format {
	case "jpeg", "png", "gif":
	default:
		return fmt.Errorf("Unsupported image format: %s", format)
	}

	return verifyDimentions(decoded)
}

// Verify an image does not exceed the preset maximum dimentions
func verifyDimentions(decoded image.Image) error {
	conf := config.Get().Images.Max
	rect := decoded.Bounds()
	if rect.Max.X-rect.Min.X > conf.Width {
		return errors.New("Image too wide")
	}
	if rect.Max.Y-rect.Min.Y > conf.Height {
		return errors.New("Image too tall")
	}
	return nil
}
