// Validation and image extraction for webm and MP4/OGG with video

package imager

import (
	"mime/multipart"
)

func processWebm(
	file multipart.File,
	fileHeader *multipart.FileHeader,
	img *ProtoImage,
) (string, error) {
	return "", nil
}
