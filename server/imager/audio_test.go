package imager

import (
	"meguca/common"
	"meguca/imager/assets"
	"testing"
)

const mp3Length uint32 = 1

func TestProcessMP3NoCover(t *testing.T) {
	t.Parallel()

	var img common.ImageCommon
	_, err := processFile(readSample(t, "sample.mp3"), &img, dummyOpts)
	if err != nil {
		t.Fatal(err)
	}

	assertLength(t, img.Length, mp3Length)
	assertFileType(t, img.FileType, common.MP3)
}

func assertFileType(t *testing.T, res, std uint8) {
	t.Helper()
	if res != std {
		t.Errorf("unexpected file type: %d : %d", std, res)
	}
}

func assertLength(t *testing.T, res, std uint32) {
	t.Helper()
	if res != std {
		t.Errorf("unexpected length: %d : %d", std, res)
	}
}

func TestProcessMP3(t *testing.T) {
	t.Parallel()

	var img common.ImageCommon
	thumb, err := processFile(readSample(t, "with_cover.mp3"), &img, dummyOpts)
	if err != nil {
		t.Fatal(err)
	}

	assertThumbnail(t, thumb)
	assertDims(t, img.Dims, assets.StdDims["png"])
	assertLength(t, img.Length, mp3Length)
}
