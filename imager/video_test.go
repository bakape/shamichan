package imager

import "testing"

func TestProcessWebm(t *testing.T) {
	cases := [...]struct {
		testName, name string
		audio          bool
		length         uint32
		dims           [4]uint16
	}{
		{
			"without sound",
			"wafel.webm", false, 5, [4]uint16{0x500, 0x2d0, 0x96, 0x54},
		},
		{
			"with sound",
			"sample.webm", true, 1, [4]uint16{0x500, 0x2d0, 0x96, 0x54},
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			res := processWebm(readSample(t, c.name))
			if res.err != nil {
				t.Fatal(res.err)
			}
			assertThumbnail(t, res.thumb)
			assertDims(t, res.dims, c.dims)
			if res.audio != c.audio {
				t.Error("unexpected audio flag value")
			}
			if res.length != c.length {
				t.Errorf("unexpected length: %d : %d", c.length, res.length)
			}
		})
	}
}
