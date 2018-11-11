package imager

import (
	"meguca/common"
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
		}, {
			name: "pdf",
			file: "sample.pdf", // Handled the same as archives
			typ:  common.PDF,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var img common.ImageCommon
			_, err := processFile(readSample(t, c.file), &img, dummyOpts)
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
		})
	}
}
