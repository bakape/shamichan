// Package assets manages imager file asset allocation and deallocation
package assets

import (
	"os"
	"path/filepath"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

const fileCreationFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL

// Only used in tests, but we still need them exported
var (
	//  StdJPEG is a JPEG sample image standard struct. Only used in tests.
	StdJPEG = common.Image{
		ImageCommon: common.ImageCommon{
			SHA1:     "012a2f912c9ee93ceb0ccb8684a29ec571990a94",
			FileType: common.JPEG,
			Dims:     StdDims["jpeg"],
			MD5:      "YOQQklgfezKbBXuEAsqopw",
			Size:     300792,
		},
		Name:    "sample.jpg",
		Spoiler: true,
	}

	// StdDims contains esulting dimentions after thumbnailing sample images.
	// Only used in tests.
	StdDims = map[string][4]uint16{
		"jpeg": {0x43c, 0x371, 0x96, 0x79},
		"png":  {0x500, 0x2d0, 0x96, 0x54},
		"gif":  {0x248, 0x2d0, 0x79, 0x96},
		"pdf":  {0x253, 0x34a, 0x69, 0x96},
	}
)

type buffer []byte

func (b buffer) append(s string) buffer {
	return append(b, s...)
}

// GetFilePaths generates file paths of the source file and its thumbnail
func GetFilePaths(SHA1 string, fileType, thumbType uint8) (paths [2]string) {
	paths[0] = SourcePath(fileType, SHA1)
	paths[1] = ThumbPath(thumbType, SHA1)
	for i := range paths {
		paths[i] = filepath.FromSlash(paths[i][1:])
	}

	return
}

func imageRoot() string {
	r := config.Get().ImageRootOverride
	if r != "" {
		return r
	}
	return "/images"
}

// ThumbPath returns the path to the thumbnail of an image
func ThumbPath(thumbType uint8, SHA1 string) string {
	root := imageRoot()
	ext := common.Extensions[thumbType]

	buf := make(buffer, 0, len(root)+len(ext)+48).
		append(root).append("/thumb/").append(SHA1)
	buf = append(buf, '.').append(ext)
	return string(buf)
}

// SourcePath returns the path to the source file on an image
func SourcePath(fileType uint8, SHA1 string) string {
	root := imageRoot()
	ext := common.Extensions[fileType]

	buf := make(buffer, 0, len(ext)+len(root)+46).
		append(root).append("/src/").append(SHA1)
	buf = append(buf, '.').append(ext)
	return string(buf)
}

// Write writes file assets to disk
func Write(name string, fileType, thumbType uint8, src, thumb []byte) error {
	data := [2][]byte{src, thumb}

	for i, path := range GetFilePaths(name, fileType, thumbType) {
		if err := writeFile(path, data[i]); err != nil {
			return err
		}
	}

	return nil
}

// Write a single file to disk with the appropriate permissions and flags
func writeFile(path string, data []byte) error {
	file, err := os.OpenFile(path, fileCreationFlags, 0660)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

// Delete deletes file assets belonging to a single upload
func Delete(name string, fileType, thumbType uint8) error {
	for _, path := range GetFilePaths(name, fileType, thumbType) {
		// Ignore somehow absent images
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// CreateDirs creates directories for processed image storage
func CreateDirs() error {
	for _, dir := range [...]string{"src", "thumb"} {
		path := filepath.Join("images", dir)
		if err := os.MkdirAll(path, 0700); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDirs recursively deletes the image storage folder. Only used for
// cleaning up after tests.
func DeleteDirs() error {
	return os.RemoveAll("images")
}

// ResetDirs removes all contents from the image storage directories. Only
// used for cleaning up after tests.
func ResetDirs() error {
	if err := DeleteDirs(); err != nil {
		return err
	}
	return CreateDirs()
}
