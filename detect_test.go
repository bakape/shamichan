package video

import "io/ioutil"
import "os"
import "testing"

func TestDetectAVFormat(t *testing.T) {
	f, err := os.Open("testdata/sample.mp4")
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	a, v, err := DecodeAVFormat(data)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("audio:", a)
	t.Log("video:", v)
}
