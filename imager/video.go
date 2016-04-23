// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	webmTool "github.com/Soreil/webm"
	"io"
)

func processWebm(file io.ReadSeeker, img *ProtoImage) error {
	return nil
}

func verifyWebm(file io.ReadSeeker) error {
	_, err := webmTool.DecodeConfig(file)
	if err != nil {
		return err
	}
	return nil
}
