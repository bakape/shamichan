package imager

import . "gopkg.in/check.v1"

const mp3Length uint32 = 1

func (*Imager) TestMP3Detection(c *C) {
	samples := [...]struct {
		ext   string
		isMP3 bool
	}{
		{"mp3", true},
		{"webm", false},
		{"txt", false},
	}

	for _, s := range samples {
		isMp3, err := detectMP3(readSample("sample."+s.ext, c))
		c.Assert(err, IsNil)
		c.Assert(isMp3, Equals, s.isMP3)
	}
}

func (*Imager) TestProcessMP3NoCover(c *C) {
	res := processMP3(readSample("sample.mp3", c))
	c.Assert(res.err, IsNil)
	c.Assert(res.thumb, DeepEquals, readSample("audio-fallback.png", c))
	c.Assert(res.dims, Equals, [4]uint16{150, 150, 150, 150})
	c.Assert(res.length, Equals, mp3Length)
}

func (*Imager) TestProcessMP3(c *C) {
	res := processMP3(readSample("with-cover.mp3", c))
	c.Assert(res.err, IsNil)
	assertThumbnail(res.thumb, c)
	c.Assert(res.dims, Equals, pngDims)
	c.Assert(res.length, Equals, mp3Length)
}
