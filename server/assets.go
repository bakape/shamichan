package server

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bakape/meguca/assets"
	"github.com/bakape/meguca/common"
	"github.com/bakape/thumbnailer/v2"
)

var (
	// Set of headers for serving images (and other uploaded files)
	imageHeaders = map[string]string{
		// max-age set to 350 days. Some caches and browsers ignore max-age, if
		// it is a year or greater, so keep it a little below.
		"Cache-Control": "max-age=30240000, public, immutable",
	}

	// For overriding during tests
	imageWebRoot = "images"
)

type fileError struct {
	name, msg string
}

func (e fileError) Error() string {
	return fmt.Sprintf("invalid file %s: %s", e.name, e.msg)
}

func newFileError(h *multipart.FileHeader, msg string) error {
	return common.StatusError{
		Err:  fileError{h.Filename, msg},
		Code: 400,
	}
}

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set separate caching policies for them.
func serveImages(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, func() (err error) {
		path := extractParam(r, "path")
		file, err := os.Open(cleanJoin(imageWebRoot, path))
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 404,
			}
		}
		defer file.Close()

		head := w.Header()
		for key, val := range imageHeaders {
			head.Set(key, val)
		}

		http.ServeContent(w, r, path, time.Time{}, file)
		return
	})
}

func cleanJoin(a, b string) string {
	return filepath.Clean(filepath.Join(a, b))
}

// Server static assets
func serveAssets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Service-Worker-Allowed", "/")
	serveFile(w, r, cleanJoin(webRoot, extractParam(r, "path")))
}

func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	handleError(w, r, func() (err error) {
		file, err := os.Open(path)
		if err != nil {
			return common.StatusError{
				Err:  err,
				Code: 404,
			}
		}
		defer file.Close()

		stats, err := file.Stat()
		if err != nil {
			return
		}
		if stats.IsDir() {
			return common.StatusError{
				Err:  errors.New("path point to directory"),
				Code: 404,
			}
		}
		modTime := stats.ModTime()
		etag := strconv.FormatInt(modTime.Unix(), 10)

		head := w.Header()
		head.Set("Cache-Control", "no-cache")
		head.Set("ETag", etag)
		http.ServeContent(w, r, path, modTime, file)
		return
	})
}

// Read a file from an asset submition form.
// If ok == false, caller should return.
func readAssetFile(w http.ResponseWriter, r *http.Request, f multipart.File,
	h *multipart.FileHeader, opts thumbnailer.Options,
) (
	out assets.File, err error,
) {
	defer f.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(f)
	if err != nil {
		return
	}
	if buf.Len() == 0 { // No file
		return
	}
	if buf.Len() > common.MaxAssetSize {
		err = newFileError(h, "too large")
		return
	}

	src, _, err := thumbnailer.Process(bytes.NewReader(buf.Bytes()), opts)
	switch {
	case err != nil:
		err = newFileError(h, err.Error())
	case src.HasAudio:
		err = newFileError(h, "has audio")
	default:
		out = assets.File{
			Data: buf.Bytes(),
			Mime: src.Mime,
		}
	}
	return
}
