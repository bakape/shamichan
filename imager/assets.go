package imager

import (
	"fmt"
	"io"
	"io/ioutil"
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

func getFilePaths(name string, fileType uint8) [3]string {
	thumbExtension := "png"
	if fileType == jpeg {
		thumbExtension = "jpg"
	}

	var paths [3]string
	paths[0] = fmt.Sprintf("src/%s.%s", name, extensions[fileType])
	paths[1] = fmt.Sprintf("thumb/%s.%s", name, thumbExtension)
	paths[2] = fmt.Sprintf("mid/%s.%s", name, thumbExtension)

	for i := range paths {
		paths[i] = filepath.FromSlash("./img/" + paths[i])
	}

	return paths
}

func writeAssets(
	name string,
	fileType uint8,
	src, thumb, mid io.Reader,
) error {
	readers := [3]io.Reader{src, thumb, mid}

	for i, path := range getFilePaths(name, fileType) {
		err := writeFile(path, readers[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path string, r io.Reader) error {
	file, err := os.OpenFile(path, fileCreationFlags, 0660)
	if err != nil {
		return err
	}
	defer file.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	_, err = file.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

func deleteAssets(name string, fileType uint8) error {
	for _, path := range getFilePaths(name, fileType) {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
