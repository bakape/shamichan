// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"fmt"
	webmTool "github.com/Soreil/webm"
	"mime/multipart"
)

func processWebm(
	file multipart.File,
	fileHeader *multipart.FileHeader,
	img *ProtoImage,
) (string, error) {
	return "", nil
}

func verifyWebm(file multipart.File) error {
	conf, err := webmTool.DecodeConfig(file)
	if err != nil {
		return err
	}
	fmt.Println(conf)
	return nil
}
