// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"io"

	webmTool "github.com/Soreil/webm"
	"github.com/bakape/meguca/types"
)

func processWebm(file io.ReadSeeker, postID int64, img *types.ProtoImage) (
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
