// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"io"

	webmTool "github.com/Soreil/video/webm"
)

func processWebm(file io.ReadSeeker) (
	io.Reader, io.Reader, error,
) {
	return nil, nil, nil
}

func verifyWebm(file io.ReadSeeker) error {
	_, err := webmTool.DecodeConfig(file)
	if err != nil {
		return err
	}
	return nil
}
