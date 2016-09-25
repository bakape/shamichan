package assets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bakape/meguca/types"
)

const fileCreationFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL

// GetFilePaths generates file paths of the source file and its thumbnail
func GetFilePaths(name string, fileType uint8) (paths [2]string) {
	thumbExtension := "png"
	if fileType == types.JPEG {
		thumbExtension = "jpg"
	}
	paths[0] = fmt.Sprintf("images/src/%s.%s", name, types.Extensions[fileType])
	paths[1] = fmt.Sprintf("images/thumb/%s.%s", name, thumbExtension)

	for i := range paths {
		paths[i] = filepath.FromSlash(paths[i])
	}

	return
}

// Write writes file assets to disk
func Write(name string, fileType uint8, src, thumb []byte) error {
	data := [2][]byte{src, thumb}

	for i, path := range GetFilePaths(name, fileType) {
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
func Delete(name string, fileType uint8) error {
	for _, path := range GetFilePaths(name, fileType) {
		if err := os.Remove(path); err != nil {
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
// cleaning up  after tests.
func DeleteDirs() error {
	return os.RemoveAll("images")
}

// ResetDirs removes all contents from the image storage directories. Only
// used for cleaning up  after tests.
func ResetDirs() error {
	if err := DeleteDirs(); err != nil {
		return err
	}
	return CreateDirs()
}
