package thumbnailer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func init() {
	SetOpts(150, 150)
}

func TestThumbnail(t *testing.T) {
	for _, ext := range [...]string{"png", "jpg", "gif", "pdf"} {
		buf := readSample(t, "sample."+ext)
		buf, w, h, err := Thumbnail(buf, ext == "jpg")
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s thumb dims: %dx%d", ext, w, h)

		var thumbExt string
		if ext == "jpg" {
			thumbExt = "jpg"
		} else {
			thumbExt = "png"
		}
		writeSample(t, fmt.Sprintf(`thumb_%s.%s`, ext, thumbExt), buf)
	}
}

func readSample(t *testing.T, name string) []byte {
	buf, err := ioutil.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

func writeSample(t *testing.T, name string, buf []byte) {
	path := filepath.Join("testdata", name)

	// Remove previous file, if any
	_, err := os.Stat(path)
	switch {
	case os.IsExist(err):
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	case os.IsNotExist(err):
	case err == nil:
	default:
		t.Fatal(err)
	}

	err = ioutil.WriteFile(path, buf, 0600)
	if err != nil {
		t.Fatal(err)
	}
}

func TestErrorPassing(t *testing.T) {
	_, _, _, err := Thumbnail(nil, false)
	if err == nil {
		t.Fatal(`expected error`)
	}
}
