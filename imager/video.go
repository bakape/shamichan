// Validation and image extraction for webm and MP4/OGG with video

package imager

import "github.com/Soreil/video"

// Extract data and thumbnail from a WebM video
func processWebm(data []byte) (res thumbResponse) {
	audio, _, err := video.DecodeAVFormat(data)
	if err != nil {
		if err.Error() == "Failed to decode audio stream" {
			err = nil
		} else {
			res.err = err
			return
		}
	}
	if audio != "" {
		res.audio = true
	}

	// TODO: Waiting on Soreil for implementation
	res.length = 60

	res.thumb, res.dims, res.err = processImage(data)
	return
}
