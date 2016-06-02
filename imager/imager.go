// Processes image files and images extracted from video, audio or PDF files

package imager

import (
	"errors"
	"fmt"
	"image"
	"io"

	"github.com/Soreil/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	"github.com/jteeuwen/imghash"
)

var (
	// Maximum dimentions for a normal small thumbnail
	normal = image.Point{X: 125, Y: 125}

	// Maximum dimentions for a high quality thumbnail
	sharp = image.Point{X: 250, Y: 250}
)

// Verify image parameters and create thumbnails
func processImage(file io.ReadSeeker, postID int64, img *types.ProtoImage) (
	large io.Reader, small io.Reader, err error,
) {
	err = verifyImage(file, postID)
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
func verifyImage(file io.ReadSeeker, postID int64) error {
	decoded, format, err := image.Decode(file)
	if err != nil {
		return util.WrapError("Error decoding image", err)
	}

	switch format {
	case "jpeg", "png", "gif":
	default:
		return fmt.Errorf("Unsupported image format: %s", format)
	}

	if err := verifyDimentions(decoded); err != nil {
		return err
	}
	return verifyUniqueness(decoded, postID)
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

// Verify an image has not been posted already recently
func verifyUniqueness(img image.Image, postID int64) error {
	res := make(chan int64)
	dedupImage <- dedupRequest{
		entry: hashEntry{
			ID:   postID,
			Hash: float64(imghash.Average(img)),
		},
		res: res,
	}
	dup := <-res
	if dup == 0 {
		return nil
	}
	return fmt.Errorf("Duplicate image of post %d", dup)
}
