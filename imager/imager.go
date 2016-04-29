// Processes image files and images extracted from video, audio or PDF files

package imager

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Soreil/apngdetector"
	_ "github.com/Soreil/imager" // TEMP
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/jteeuwen/imghash"
	"image"
	"io"
	"io/ioutil"
)

var (
	// Maximum dimentions for a normal small thumbnail
	normal = image.Point{X: 125, Y: 125}

	// Maximum dimentions for a high quality thumbnail
	sharp = image.Point{X: 250, Y: 250}
)

// Verify image parameters and create thumbnails
func processImage(file io.Reader, img *ProtoImage) error {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return util.WrapError("Error reading from file", err)
	}
	buf := bytes.NewBuffer(data)
	if err := verifyImage(buf, img.PostID); err != nil {
		return err
	}
	if img.FileType == png {
		img.APNG = apngdetector.Detect(data)
	}

	// These hashes will already be calculated, if the source file is a video,
	// MP3, etc.
	if img.SHA1 == "" {
		hashFile(data, img)
	}

	return nil
}

// Verify image dimentions and that it has not been posted before in the
// configured time
func verifyImage(buf io.Reader, postID uint64) error {
	decoded, format, err := image.Decode(buf)
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
	if err := verifyUniqueness(decoded, postID); err != nil {
		return err
	}
	return nil
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
func verifyUniqueness(img image.Image, postID uint64) error {
	res := make(chan uint64)
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

// Calculate SHA1 and MD5 hashes for a file
func hashFile(data []byte, img *ProtoImage) {
	sha1Hash := sha1.Sum(data)
	md5Hash := md5.Sum(data)
	img.SHA1 = hex.EncodeToString(sha1Hash[:])
	img.MD5 = hex.EncodeToString(md5Hash[:])
}
