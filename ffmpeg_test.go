package goffmpeg

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
