package video

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAVFormat(t *testing.T) {
	t.Parallel()
	f := openFile(t)
	defer f.Close()
	a, v, err := DecodeAVFormat(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("audio:", a)
	t.Log("video:", v)
}

func openFile(t *testing.T) *os.File {
	f, err := os.Open(filepath.Join("testdata", "sample.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestDetectAVFormatDetail(t *testing.T) {
	t.Parallel()
	f := openFile(t)
	defer f.Close()
	a, v, err := DecodeAVFormatDetail(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("audio:", a)
	t.Log("video:", v)
}

func TestDecode(t *testing.T) {
	t.Parallel()
	f := openFile(t)
	defer f.Close()
	_, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecodeConfig(t *testing.T) {
	t.Parallel()
	f := openFile(t)
	defer f.Close()
	img, err := DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v\n", img)
}

func TestDecodeLength(t *testing.T) {
	t.Parallel()
	f := openFile(t)
	defer f.Close()
	l, err := DecodeLength(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(l)
}
