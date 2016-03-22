// Processes image files and images extracted from video, audio or PDF files

package imager

import (
	"mime/multipart"
)

func processImage(
	file multipart.File,
	fileHeader *multipart.FileHeader,
	img *ProtoImage,
) (string, error) {
	return "", nil
}
