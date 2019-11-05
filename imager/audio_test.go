package imager

import (
	"testing"

	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/imager/assets"
	"github.com/Chiiruno/meguca/test"
)

const mp3Length uint32 = 1

func TestProcessMP3NoCover(t *testing.T) {
	t.Parallel()

	var img common.ImageCommon
	f := test.OpenSample(t, "sample.mp3")
	defer f.Close()
	_, err := processFile(f, &img, dummyOpts)
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
	f := test.OpenSample(t, "with_cover.mp3")
	defer f.Close()
	thumb, err := processFile(f, &img, dummyOpts)
	if err != nil {
		t.Fatal(err)
	}

	assertThumbnail(t, thumb)
	assertDims(t, img.Dims, assets.StdDims["png"])
	assertLength(t, img.Length, mp3Length)
}
