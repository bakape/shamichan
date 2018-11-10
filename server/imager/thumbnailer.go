package imager

import "github.com/bakape/thumbnailer"

const mimePDF = "application/pdf"

func init() {
	for _, fn := range [...]thumbnailer.MatcherFunc{
		detect7z,
		detectZip,
		detectTarGZ,
		detectTarXZ,
		detectText, // Has to be last, in case any other formats are pure UTF-8
	} {
		thumbnailer.RegisterMatcher(fn)
	}
	for _, m := range [...]string{
		mimeZip, mime7Zip, mimeTarGZ, mimeTarXZ, mimeText,
		/// PDF thumbnailing can be very buggy and ghostcript is unreliable.
		mimePDF,
	} {
		thumbnailer.RegisterProcessor(m, noopProcessor)
	}
}

// Does nothing.
// Needed for the thumbnailer to accept these as validly processed.
func noopProcessor(src thumbnailer.Source, _ thumbnailer.Options) (
	thumbnailer.Source, thumbnailer.Thumbnail, error,
) {
	return src, thumbnailer.Thumbnail{}, nil
}
