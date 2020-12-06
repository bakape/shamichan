// Package assets manages imager file asset allocation and deallocation
package assets

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/bakape/meguca/imager/common"
)

// GetFilePaths generates file paths of the source file and its thumbnail
func GetFilePaths(
	SHA1 common.SHA1Hash,
	fileType, thumbType common.FileType,
) (paths [2]string) {
	paths[0] = fmt.Sprintf(
		"/images/src/%s.%s",
		SHA1,
		common.Extensions[fileType],
	)
	paths[1] = fmt.Sprintf(
		"/images/thumb/%s.%s",
		SHA1,
		common.Extensions[thumbType],
	)
	for i := range paths {
		paths[i] = filepath.FromSlash(paths[i][1:])
	}

	return
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
func Write(
	SHA1 common.SHA1Hash,
	fileType, thumbType common.FileType,
	src, thumb io.ReadSeeker,
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

	var ch chan error
	if thumb != nil { // Archives, audio, etc. can be missing thumbnails
		ch = make(chan error, 1) // Buffered to not leak on early return
		go func() {
			ch <- writeFile(paths[1], thumb)
		}()
	}
	err = writeFile(paths[0], src)
	if err != nil {
		return
	}
	if thumb != nil {
		err = <-ch
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
func Delete(SHA1 common.SHA1Hash, fileType, thumbType common.FileType) error {
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
		if err := os.MkdirAll(path, 0705); err != nil {
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
