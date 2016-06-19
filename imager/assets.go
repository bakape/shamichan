package imager

import (
	"fmt"
	"os"
	"path/filepath"
)

const fileCreationFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL

var extensions = map[uint8]string{
	jpeg: "jpg",
	png:  "png",
	gif:  "gif",
	webm: "webm",
	pdf:  "pdf",
}

// Generate file paths of the source file and its thumbnail
func getFilePaths(name string, fileType uint8) (paths [2]string) {
	thumbExtension := "png"
	if fileType == jpeg {
		thumbExtension = "jpg"
	}
	paths[0] = fmt.Sprintf("images/src/%s.%s", name, extensions[fileType])
	paths[1] = fmt.Sprintf("images/thumb/%s.%s", name, thumbExtension)

	for i := range paths {
		paths[i] = filepath.FromSlash(paths[i])
	}

	return
}

// Write file assets to disk
func writeAssets(name string, fileType uint8, src, thumb []byte) error {
	data := [2][]byte{src, thumb}

	for i, path := range getFilePaths(name, fileType) {
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

// Delete file assets belonging to a single upload
func deleteAssets(name string, fileType uint8) error {
	for _, path := range getFilePaths(name, fileType) {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
