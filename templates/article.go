package templates

import (
	"fmt"
	"time"

	"github.com/bakape/meguca/types"
)

// Returns the HTTP path to the thumbnail of an image
func thumbPath(img types.Image) string {
	var ext string
	if img.FileType == types.JPEG {
		ext = "jpg"
	} else {
		ext = "png"
	}
	return fmt.Sprintf("/images/thumb/%s.%s", img.SHA1, ext)
}

// Returns the HTTP path to the source file
func sourcePath(img types.Image) string {
	return fmt.Sprintf(
		"/images/src/%s.%s",
		img.SHA1,
		types.Extensions[img.FileType],
	)
}

func extension(fileType uint8) string {
	return types.Extensions[fileType]
}

// Renders the post creation time field
func renderTime(sec int64) string {
	return time.Unix(sec, 0).Format("2 Jan 2006 (Mon) 15:04")
}

// Renders a human-readable representation video/audio length
func readableLength(l uint32) string {
	if l < 60 {
		return fmt.Sprintf("0:%02d", l)
	}
	min := l / 60
	return fmt.Sprintf("%02d:%02d", min, l-min)
}

// Renders a human-readable representation of file size
func readableFileSize(s int) string {
	if s < (1 << 10) {
		return fmt.Sprintf("%d B", s)
	}
	if s < (1 << 20) {
		return fmt.Sprintf("%d KB", s/(1<<10))
	}
	return fmt.Sprintf("%.1f MB", float32(s)/(1<<20))
}
