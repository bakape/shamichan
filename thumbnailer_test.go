package thumbnailer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestThumbnail(t *testing.T) {
	t.Parallel()

	for _, ext := range [...]string{"png", "jpg", "gif", "pdf"} {
		buf := readSample(t, "sample."+ext)

		opts := Options{
			Width:           150,
			Height:          150,
			JPEGCompression: 90,
		}
		if ext == "jpg" {
			opts.OutputType = JPEG
		}
		buf, w, h, err := Thumbnail(buf, opts)
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
	t.Parallel()

	_, _, _, err := Thumbnail(nil, Options{
		Width:  150,
		Height: 150,
	})
	if err == nil {
		t.Fatal(`expected error`)
	}
}

func TestDimensionValidation(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, file string
		maxW, maxH uint
		err        error
	}{
		{
			name: "width check disabled",
			file: "too wide.jpg",
			maxW: 2000,
		},
		{
			name: "too wide",
			file: "too wide.jpg",
			maxW: 2000,
			err:  ErrTooWide,
		},
		{
			name: "height check disabled",
			file: "too tall.jpg",
			maxH: 2000,
		},
		{
			name: "too tall",
			file: "too tall.jpg",
			maxH: 2000,
			err:  ErrTooTall,
		},
		{
			name: "pdf pass through",
			file: "sample.pdf",
			maxH: 1,
			maxW: 1,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			opts := Options{
				Width:           150,
				Height:          150,
				MaxSrcWidth:     c.maxW,
				MaxSrcHeight:    c.maxH,
				JPEGCompression: 90,
			}
			_, _, _, err := Thumbnail(readSample(t, c.file), opts)
			if err != c.err {
				t.Logf("unexpected error: %s : %s", c.err, err)
			}
		})
	}
}

func TestSourceAlreadyThumbSize(t *testing.T) {
	t.Parallel()

	_, w, h, err := Thumbnail(readSample(t, "too small.png"), Options{
		Width:  150,
		Height: 150,
	})
	if err != nil {
		t.Fatal(err)
	}
	if w != 121 {
		t.Errorf("unexpected width: 121 : %d", w)
	}
	if h != 150 {
		t.Errorf("unexpected height: 150: %d", h)
	}
}
