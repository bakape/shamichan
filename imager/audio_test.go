package imager

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
)

const mp3Length uint32 = 1

func TestMP3Detection(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		ext   string
		isMP3 bool
	}{
		{"mp3", true},
		{"webm", false},
		{"txt", false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.ext, func(t *testing.T) {
			t.Parallel()

			isMp3, err := detectMP3(readSample(t, "sample."+c.ext))
			if err != nil {
				t.Fatal(err)
			}
			if isMp3 != c.isMP3 {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestProcessMP3NoCover(t *testing.T) {
	res := processMP3(readSample(t, "sample.mp3"))
	if res.err != nil {
		t.Fatal(res.err)
	}

	AssertBufferEquals(t, res.thumb, readFallbackThumb(t, "audio-fallback.png"))
	assertDims(t, res.dims, [4]uint16{150, 150, 150, 150})
	if res.length != mp3Length {
		t.Fatalf("unexpected length: %d : %d", mp3Length, res.length)
	}
}

func TestProcessMP3(t *testing.T) {
	res := processMP3(readSample(t, "with-cover.mp3"))
	if res.err != nil {
		t.Fatal(res.err)
	}

	assertThumbnail(t, res.thumb)
	assertDims(t, res.dims, assets.StdDims["png"])
	if res.length != mp3Length {
		t.Fatalf("unexpected length: %d : %d", mp3Length, res.length)
	}
}

func readFallbackThumb(t *testing.T, name string) []byte {
	buf, err := ioutil.ReadFile(filepath.Join(assetRoot, name))
	if err != nil {
		t.Fatal(err)
	}
	return buf
}
