// Package assets manages imager file asset allocation and deallocation
package assets

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
)

// Only used in tests, but we still need them exported
var (
	//  StdJPEG is a JPEG sample image standard struct. Only used in tests.
	StdJPEG = common.Image{
		ImageCommon: common.ImageCommon{
			Video:     true,
			SHA1:      "012a2f912c9ee93ceb0ccb8684a29ec571990a94",
			FileType:  common.JPEG,
			ThumbType: common.WEBP,
			Dims:      StdDims["jpeg"],
			MD5:       "YOQQklgfezKbBXuEAsqopw",
			Size:      300792,
		},
		Name: "sample.jpg",
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

// GetFilePaths generates file paths of the source file and its thumbnail
func GetFilePaths(SHA1 string, fileType, thumbType uint8) (paths [2]string) {
	paths[0] = util.ConcatStrings(
		"/images/src/",
		SHA1,
		".",
		common.Extensions[fileType],
	)
	paths[1] = util.ConcatStrings(
		"/images/thumb/",
		SHA1,
		".",
		common.Extensions[thumbType],
	)
	for i := range paths {
		paths[i] = filepath.FromSlash(paths[i][1:])
	}

	return
}

// RelativeSourcePath returns a file's source path relative to the root path
func RelativeSourcePath(fileType uint8, SHA1 string) string {
	return util.ConcatStrings(
		"/assets/images/src/",
		SHA1,
		".",
		common.Extensions[fileType],
	)
}

// RelativeThumbPath returns a thumbnail's path relative to the root path
func RelativeThumbPath(thumbType uint8, SHA1 string) string {
	return util.ConcatStrings(
		"/assets/images/thumb/",
		SHA1,
		".",
		common.Extensions[thumbType],
	)
}

// ImageSearchPath returns the relative path used for image search file lookups.
// If files is not JPEG, PNG or GIF, returns the thumbnail instead of the source
// file.
func ImageSearchPath(img common.ImageCommon) string {
	switch img.FileType {
	case common.JPEG, common.PNG, common.GIF:
		if img.Size < 8<<20 {
			return RelativeSourcePath(img.FileType, img.SHA1)
		}
	}
	return RelativeThumbPath(img.ThumbType, img.SHA1)
}

func imageRoot() string {
	r := config.Get().ImageRootOverride
	if r != "" {
		return r
	}
	return "/assets/images"
}

// ThumbPath returns the path to the thumbnail of an image
func ThumbPath(thumbType uint8, SHA1 string) string {
	return util.ConcatStrings(
		imageRoot(),
		"/thumb/",
		SHA1,
		".",
		common.Extensions[thumbType],
	)
}

// SourcePath returns the path to the source file on an image
func SourcePath(fileType uint8, SHA1 string) string {
	return util.ConcatStrings(
		imageRoot(),
		"/src/",
		SHA1,
		".",
		common.Extensions[fileType],
	)
}

// Return free space on image storage device.
// Image source file and thumbnail directories must be on the same drive.
func freeSpace() (n uint64, err error) {
	var stats syscall.Statfs_t
	path, err := filepath.Abs("images/src")
	if err != nil {
		return
	}
	err = syscall.Statfs(path, &stats)
	return stats.Bavail * uint64(stats.Bsize), err
}

// Write writes file assets to disk
func Write(SHA1 string, fileType, thumbType uint8, src, thumb io.ReadSeeker,
) (
	err error,
) {
	// Assert at least 100 MB of free disk space is available
	if !common.IsCI {
		var free uint64
		free, err = freeSpace()
		if err != nil {
			return
		}
		if free < (100 << 20) {
			return errors.New("not enough disk space")
		}
	}

	paths := GetFilePaths(SHA1, fileType, thumbType)

	// Don't write files in parallel to reduce the amount of threads the Go
	// runtime needs to spawn.
	err = writeFile(paths[0], src)
	if err != nil {
		return
	}
	if thumb != nil { // Archives, audio, etc. can be missing thumbnails
		err = writeFile(paths[1], thumb)
	}
	return
}

// Write a single file to disk with the appropriate permissions and flags
func writeFile(path string, src io.ReadSeeker) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = src.Seek(0, 0)
	if err != nil {
		return
	}
	_, err = io.Copy(file, src)
	return
}

// Delete deletes file assets belonging to a single upload
func Delete(SHA1 string, fileType, thumbType uint8) error {
	for _, path := range GetFilePaths(SHA1, fileType, thumbType) {
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
