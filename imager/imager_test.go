package imager

import (
	"image"
	"image/jpeg"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Soreil/imager"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
)

func TestMain(m *testing.M) {
	db.DBName = "meguca_test_imager"
	db.IsTest = true
	assetRoot = "testdata"
	config.Set(config.Configs{})
	if err := db.LoadDB(); err != nil {
		panic(err)
	}
	if err := assets.CreateDirs(); err != nil {
		panic(err)
	}
	defer assets.DeleteDirs()

	code := m.Run()

	os.Exit(code)
}

func resetDirs(t *testing.T) {
	if err := assets.ResetDirs(); err != nil {
		t.Fatal(err)
	}
}

func readSample(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func openFile(t *testing.T, name string) *os.File {
	f, err := os.Open(filepath.FromSlash("testdata/" + name))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// How do we assert a thumbnail?
func assertThumbnail(t *testing.T, thumb []byte) {
	if len(thumb) < 100 {
		t.Fatal("thumbnail too small")
	}
}

func assertDims(t *testing.T, res, std [4]uint16) {
	if res != std {
		LogUnexpected(t, std, res)
	}
}

func TestInitImager(t *testing.T) {
	config.Set(config.Configs{
		JPEGQuality: 90,
		PNGQuality:  20,
	})
	if err := InitImager(); err != nil {
		t.Fatal(err)
	}

	jpegSTD := jpeg.Options{Quality: 90}
	if imager.JPEGOptions != jpegSTD {
		LogUnexpected(t, jpegSTD, imager.JPEGOptions)
	}
	if imager.PNGQuantization != 20 {
		LogUnexpected(t, 20, imager.PNGQuantization)
	}
}

func TestVerifyDimentions(t *testing.T) {
	config.Set(config.Configs{
		MaxWidth:  2000,
		MaxHeight: 2000,
	})

	cases := [...]struct {
		testName, name string
		err            error
		dims           [4]uint16
	}{
		{"too wide", "too wide.jpg", errTooWide, [4]uint16{2001, 720, 0, 0}},
		{"too tall", "too tall.jpg", errTooTall, [4]uint16{1280, 2001, 0, 0}},
		{"pass", "sample.jpg", nil, [4]uint16{1084, 881, 0, 0}},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			file := openFile(t, c.name)
			defer file.Close()
			img, _, err := image.Decode(file)
			if err != nil {
				t.Fatal(err)
			}
			dims, err := verifyDimentions(img)
			if err != c.err {
				UnexpectedError(t, err)
			}
			assertDims(t, dims, c.dims)
		})
	}
}

func TestImageProcessing(t *testing.T) {
	config.Set(config.Configs{
		MaxWidth:  2000,
		MaxHeight: 2000,
	})

	cases := [...]struct {
		ext  string
		dims [4]uint16
	}{
		{"jpg", assets.StdDims["jpeg"]},
		{"png", assets.StdDims["png"]},
		{"gif", assets.StdDims["gif"]},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.ext, func(t *testing.T) {
			t.Parallel()

			thumb, dims, err := processImage(readSample(t, "sample."+c.ext))
			if err != nil {
				t.Fatal(err)
			}
			assertThumbnail(t, thumb)
			assertDims(t, dims, c.dims)
		})
	}
}
