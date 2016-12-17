package imager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/imager/assets"
)

func TestImageProcessing(t *testing.T) {
	config.Set(config.Configs{
		MaxWidth:    2000,
		MaxHeight:   2000,
		JPEGQuality: 80,
	})

	cases := [...]struct {
		ext   string
		dims  [4]uint16
		isPNG bool
	}{
		{"jpg", assets.StdDims["jpeg"], false},
		{"png", assets.StdDims["png"], true},
		{"gif", assets.StdDims["gif"], true},
		{"pdf", assets.StdDims["pdf"], false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.ext, func(t *testing.T) {
			t.Parallel()

			buf := readSample(t, "sample."+c.ext)
			thumb, dims, isPNG, err := processImage(buf)
			if err != nil {
				t.Fatal(err)
			}

			assertThumbnail(t, thumb)
			assertDims(t, dims, c.dims)

			if isPNG != c.isPNG {
				t.Error("unexpected thumbnail type")
			}

			var thumbExt string
			if isPNG {
				thumbExt = "png"
			} else {
				thumbExt = "jpg"
			}
			t.Logf(`dims: %dx%d`, dims[2], dims[3])
			writeSample(t, fmt.Sprintf("thumb_%s.%s", c.ext, thumbExt), thumb)
		})
	}
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

func TestGraphicsMagicErrorPassing(t *testing.T) {
	config.Set(config.Configs{
		MaxWidth:  2000,
		MaxHeight: 2000,
	})
	_, _, _, err := processImage(nil)
	if err == nil {
		t.Fatal(`expected error`)
	}
}

func TestDimensionValidation(t *testing.T) {
	config.Set(config.Configs{
		MaxWidth:  2000,
		MaxHeight: 2000,
	})

	cases := [...]struct {
		name, file string
		err        error
	}{
		{
			name: "too wide",
			file: "too wide.jpg",
			err:  errTooWide,
		},
		{
			name: "too tall",
			file: "too tall.jpg",
			err:  errTooTall,
		},
		{
			name: "pass",
			file: "sample.jpg",
		},
		{
			name: "pdf pass through",
			file: "sample.pdf",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			_, _, _, err := processImage(readSample(t, c.file))

			if err != c.err {
				t.Fatalf("unexpected error: `%s` : `%s`", c.err, err)
			}
		})
	}
}

func TestSourceAlreadyThumbSize(t *testing.T) {
	config.Set(config.Configs{
		MaxWidth:  2000,
		MaxHeight: 2000,
	})

	_, dims, _, err := processImage(readSample(t, "too small.png"))

	assertDims(t, dims, [4]uint16{121, 150, 121, 150})
	if err != nil {
		t.Fatal(err)
	}
}
