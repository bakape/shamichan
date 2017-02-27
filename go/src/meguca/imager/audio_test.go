package imager

import (
	"meguca/common"
	"meguca/imager/assets"
	. "meguca/test"
	"testing"
)

const mp3Length uint32 = 1

func TestProcessMP3NoCover(t *testing.T) {
	t.Parallel()

	thumb, img, err := processFile(
		readSample(t, "sample.mp3"),
		common.ImageCommon{},
		dummyOpts,
	)
	if err != nil {
		t.Fatal(err)
	}

	assertLength(t, img.Length, mp3Length)
	assertFileType(t, img.FileType, common.MP3)
	AssertBufferEquals(t, thumb, readFallbackThumb(t, "audio.png"))
	assertDims(t, img.Dims, [4]uint16{150, 150, 150, 150})
}

func assertFileType(t *testing.T, res, std uint8) {
	if res != std {
		t.Errorf("unexpected file type: %d : %d", std, res)
	}
}

func assertLength(t *testing.T, res, std uint32) {
	if res != std {
		t.Errorf("unexpected length: %d : %d", std, res)
	}
}

func TestProcessMP3(t *testing.T) {
	t.Parallel()

	thumb, img, err := processFile(
		readSample(t, "with_cover.mp3"),
		common.ImageCommon{},
		dummyOpts,
	)
	if err != nil {
		t.Fatal(err)
	}

	assertThumbnail(t, thumb)
	assertDims(t, img.Dims, assets.StdDims["png"])
	assertLength(t, img.Length, mp3Length)
}

func readFallbackThumb(t *testing.T, name string) []byte {
	buf, err := Asset(name)
	if err != nil {
		t.Fatal(err)
	}
	return buf
}
