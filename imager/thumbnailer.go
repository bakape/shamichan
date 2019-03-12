package imager

import (
	"image"
	"io"

	"github.com/bakape/thumbnailer"
)

const mimePDF = "application/pdf"

func init() {
	for _, fn := range [...]thumbnailer.MatcherFunc{
		detectTarGZ,
		detectTarXZ,
		detectText, // Has to be last, in case any other formats are pure UTF-8
	} {
		thumbnailer.RegisterMatcher(fn)
	}
	for _, m := range [...]string{
		mime7Zip, mimeTarGZ, mimeTarXZ, mimeText,
		/// PDF thumbnailing can be very buggy and ghostcript is unreliable and
		// a security risk
		mimePDF,
	} {
		thumbnailer.RegisterProcessor(m, noopProcessor)
	}
}

// Does nothing.
// Needed for the thumbnailer to accept these as validly processed.
func noopProcessor(rs io.ReadSeeker, _ *thumbnailer.Source,
	_ thumbnailer.Options,
) (
	image.Image, error,
) {
	return nil, thumbnailer.ErrCantThumbnail
}
