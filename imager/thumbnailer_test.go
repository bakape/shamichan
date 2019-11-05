package imager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/imager/assets"
	"github.com/Chiiruno/meguca/test"

	"github.com/bakape/thumbnailer/v2"
)

func TestImageProcessing(t *testing.T) {
	config.Set(config.Configs{
		MaxWidth:  2000,
		MaxHeight: 2000,
	})

	cases := [...]struct {
		noThumb    bool
		name, file string
		dims       [4]uint16
	}{
		{
			name: "jpg",
			file: "sample.jpg",
			dims: assets.StdDims["jpeg"],
		},
		{
			name: "png",
			file: "sample.png",
			dims: assets.StdDims["png"],
		},
		{
			name: "webp",
			file: "sample.webp",
			dims: assets.StdDims["png"],
		},
		{
			name: "gif",
			file: "sample.gif",
			dims: assets.StdDims["gif"],
		},
		{
			name:    "invalid UTF-8 in metdata",
			file:    "invalid_utf8.mp3",
			dims:    [4]uint16{},
			noThumb: true,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var img common.ImageCommon
			f := test.OpenSample(t, c.file)
			defer f.Close()
			thumb, err := processFile(f, &img, thumbnailer.Options{
				ThumbDims: thumbnailer.Dims{
					Width:  150,
					Height: 150,
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			assertThumbnail(t, thumb)
			assertDims(t, img.Dims, c.dims)

			ft := common.WEBP
			if c.noThumb {
				ft = common.NoFile
			}
			assertFileType(t, img.ThumbType, ft)

			for _, s := range [...]string{img.Artist, img.Title} {
				test.AssertEquals(t, utf8.ValidString(s), true)
			}

			t.Logf(`dims: %dx%d`, img.Dims[2], img.Dims[3])
			writeSample(t, fmt.Sprintf("thumb_%s.webp", c.file), thumb)
		})
	}
}

func writeSample(t *testing.T, name string, buf []byte) {
	t.Helper()

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
