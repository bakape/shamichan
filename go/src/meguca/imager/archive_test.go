package imager

import (
	"meguca/common"
	. "meguca/test"
	"strings"
	"testing"

	"github.com/bakape/thumbnailer"
)

var (
	dummyOpts = thumbnailer.Options{
		ThumbDims: thumbnailer.Dims{
			Width:  150,
			Height: 150,
		},
	}
)

func TestProcessArchive(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, file, err string
		typ             uint8
	}{
		{
			name: "ZIP",
			file: "sample.zip",
			typ:  common.ZIP,
		},
		{
			name: "7zip",
			file: "sample.7z",
			typ:  common.SevenZip,
		},
		{
			name: "tar.gz",
			file: "sample.tar.gz",
			typ:  common.TGZ,
		},
		{
			name: "tar.xz",
			file: "sample.tar.xz",
			typ:  common.TXZ,
		},
		{
			name: "file too small",
			file: "sample.txt",
			err:  "unsupported MIME type",
		},
	}

	fallback := readFallbackThumb(t, "archive.png")

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			thumb, img, err := processFile(
				readSample(t, c.file),
				common.ImageCommon{},
				dummyOpts,
			)

			if c.err != "" {
				if err == nil {
					t.Fatalf("expected an error")
				}
				if !strings.HasPrefix(err.Error(), c.err) {
					t.Fatalf("unexpected error: %#v", err)
				}
				return
			} else if err != nil {
				t.Fatal(err)
			}

			assertFileType(t, img.FileType, c.typ)
			AssertBufferEquals(t, thumb, fallback)
			assertDims(t, img.Dims, [4]uint16{150, 150, 150, 150})
		})
	}
}
