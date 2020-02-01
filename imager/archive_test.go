package imager

// var (
// 	dummyOpts = thumbnailer.Options{
// 		ThumbDims: thumbnailer.Dims{
// 			Width:  150,
// 			Height: 150,
// 		},
// 	}
// )

// func TestProcessArchive(t *testing.T) {
// 	t.Parallel()

// 	cases := [...]struct {
// 		name, file, err string
// 		typ             common.FileType
// 		hasThumb        bool
// 	}{
// 		{
// 			name:     "ZIP",
// 			file:     "sample.zip",
// 			typ:      common.CBZ,
// 			hasThumb: true,
// 		},
// 		{
// 			name:     "RAR",
// 			file:     "sample.rar",
// 			typ:      common.CBR,
// 			hasThumb: true,
// 		},
// 		{
// 			name: "7zip",
// 			file: "sample.7z",
// 			typ:  common.SevenZip,
// 		},
// 		{
// 			name: "tar.gz",
// 			file: "sample.tar.gz",
// 			typ:  common.TGZ,
// 		},
// 		{
// 			name: "tar.xz",
// 			file: "sample.tar.xz",
// 			typ:  common.TXZ,
// 		},
// 		{
// 			name: "pdf",
// 			file: "sample.pdf", // Handled the same as archives
// 			typ:  common.PDF,
// 		},
// 	}

// 	for i := range cases {
// 		c := cases[i]
// 		t.Run(c.name, func(t *testing.T) {
// 			t.Parallel()

// 			var img common.ImageCommon
// 			f := test.OpenSample(t, c.file)
// 			defer f.Close()
// 			thumb, err := processFile(f, &img, dummyOpts)
// 			if c.err != "" {
// 				if err == nil {
// 					t.Fatalf("expected an error")
// 				}
// 				if !strings.HasPrefix(err.Error(), c.err) {
// 					t.Fatalf("unexpected error: %#v", err)
// 				}
// 				return
// 			} else if err != nil {
// 				t.Fatal(err)
// 			}
// 			hasThumb := len(thumb) != 0
// 			if hasThumb != c.hasThumb {
// 				t.Fatalf("unexpected thumbnail generation: %t", hasThumb)
// 			}

// 			assertFileType(t, img.FileType, c.typ)
// 		})
// 	}
// }
