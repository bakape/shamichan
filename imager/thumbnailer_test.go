package imager

import (
	"fmt"
	"io/ioutil"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/test"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakape/thumbnailer"
)

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
		{"webp", assets.StdDims["png"]},
		{"gif", assets.StdDims["gif"]},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.ext, func(t *testing.T) {
			t.Parallel()

			var img common.ImageCommon
			f := test.OpenSample(t, "sample."+c.ext)
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
			assertFileType(t, img.ThumbType, common.WEBP)

			t.Logf(`dims: %dx%d`, img.Dims[2], img.Dims[3])
			writeSample(t, fmt.Sprintf("thumb_%s.webp", c.ext), thumb)
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
