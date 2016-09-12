package imager

import (
	. "gopkg.in/check.v1"
)

func (*Imager) TestProcessWebm(c *C) {
	samples := [...]struct {
		name   string
		audio  bool
		length uint32
		dims   [4]uint16
	}{
		{"wafel.webm", false, 5, [4]uint16{0x500, 0x2d0, 0x96, 0x54}},
		{"sample.webm", true, 1, [4]uint16{0x500, 0x2d0, 0x96, 0x54}},
	}

	for _, s := range samples {
		res := processWebm(readSample(s.name, c))
		c.Assert(res.err, IsNil)
		assertThumbnail(res.thumb, c)
		c.Assert(res.dims, Equals, s.dims)
		c.Assert(res.audio, Equals, s.audio)
		c.Assert(res.length, Equals, s.length)
	}
}