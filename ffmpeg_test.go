package video

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

var extensions = []string{"mp4", "webm", "mkv"}


// func TestDecoder(t *testing.T) {
// 	t.Parallel()

// 	for i := range extensions {
// 		ext := extensions[i]
// 		t.Run(ext, func(t *testing.T) {
// 			t.Parallel()

// 			f := openSample(t, ext)
// 			defer f.Close()

// 			d, err := NewDecoder(f)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			defer d.Close()

// 			a, v, err := d.AVFormat(false)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			t.Log("audio:", a)
// 			t.Log("video:", v)

// 			a, v, err = d.AVFormat(true)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			t.Log("audio:", a)
// 			t.Log("video:", v)

// 			_, err = d.Thumbnail()
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			img, err := d.Config()
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			t.Logf("%#v\n", img)

// 			l, err := d.Length()
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			t.Log(l)
// 		})
// 	}
// }

func openSample(t *testing.T, ext string) *os.File {
	f, err := os.Open(samplePath(ext))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func samplePath(ext string) string {
	return filepath.Join("testdata", "sample."+ext)
}

func TestDecode(t *testing.T) {
	t.Parallel()

	dests := [...]string{"png", "jpg"}
	for i := range dests {
		dest := dests[i]
		t.Run("to "+dest, func(t *testing.T) {
			t.Parallel()

			for i := range extensions {
				ext := extensions[i]
				t.Run(ext, func(t *testing.T) {
					t.Parallel()

					filename := samplePath(ext)
					if _, err := os.Stat(filename); err != nil {
						t.Fatal(err)
					}
					file, err := os.Open(filename)
					defer file.Close()
					if err != nil {
						t.Fatal(err)
					}

					img, _, err := image.Decode(file)
					if err != nil {
						t.Fatal(err)
					}

					out, err := os.Create(filename + "." + dest)
					if err != nil {
						t.Fatal(err)
					}
					if err := jpeg.Encode(out, img, nil); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

func TestDecodeConfig(t *testing.T) {
	t.Parallel()

	for i := range extensions {
		ext := extensions[i]
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

			filename := samplePath(ext)
			if _, err := os.Stat(filename); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(filename)
			defer file.Close()
			if err != nil {
				t.Fatal(err)
			}

			img, name, err := image.DecodeConfig(file)
			if err != nil {
				t.Fatal(err)
			}
			t.Log(img, name)
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	for i := range extensions {
		ext := extensions[i]
		b.Run(ext, func(b *testing.B) {
			filename := samplePath(ext)
			if _, err := os.Stat(filename); err != nil {
				b.Fatal(err)
			}

			for i := 0; i < b.N; i++ {
				file, err := os.Open(filename)
				if err != nil {
					b.Fatal(err)
				}
				_, err = Decode(file)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
