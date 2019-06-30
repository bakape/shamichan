package server

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bakape/meguca/assets"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/thumbnailer"
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
	path := extractParam(r, "path")
	file, err := os.Open(cleanJoin(imageWebRoot, path))
	if err != nil {
		text404(w)
		return
	}
	defer file.Close()

	head := w.Header()
	for key, val := range imageHeaders {
		head.Set(key, val)
	}

	http.ServeContent(w, r, path, time.Time{}, file)
}

func cleanJoin(a, b string) string {
	return filepath.Clean(filepath.Join(a, b))
}

// Server static assets
func serveAssets(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.RequestURI, "worker.js") {
		w.Header().Set("Service-Worker-Allowed", "/")
	}
	serveFile(w, r, cleanJoin(webRoot, extractParam(r, "path")))
}

func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		text404(w)
		return
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		httpError(w, r, err)
		return
	}
	if stats.IsDir() {
		text404(w)
		return
	}
	modTime := stats.ModTime()
	etag := strconv.FormatInt(modTime.Unix(), 10)

	head := w.Header()
	head.Set("Cache-Control", "no-cache")
	head.Set("ETag", etag)
	http.ServeContent(w, r, path, modTime, file)
}

// Set the banners of a board
func setBanners(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		board, err := parseAssetForm(w, r, common.MaxNumBanners)
		if err != nil {
			return
		}

		var (
			opts = thumbnailer.Options{
				MaxSourceDims: thumbnailer.Dims{
					Width:  300,
					Height: 100,
				},
				ThumbDims: thumbnailer.Dims{
					Width:  300,
					Height: 100,
				},
				AcceptedMimeTypes: map[string]bool{
					"image/jpeg": true,
					"image/png":  true,
					"image/gif":  true,
					"video/webm": true,
				},
			}
			banners = make([]assets.File, 0, common.MaxNumBanners)
			files   = r.MultipartForm.File["banners"]
			file    multipart.File
			h       *multipart.FileHeader
			out     assets.File
		)

		for i := 0; i < common.MaxNumBanners && i < len(files); i++ {
			h = files[i]
			file, err = h.Open()
			if err != nil {
				err = newFileError(h, err.Error())
				return
			}

			out, err = readAssetFile(w, r, file, h, opts)
			if err != nil {
				return
			}
			banners = append(banners, out)
		}

		return db.SetBanners(board, banners)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Parse form for uploading file assets for a board.
// maxSize specifies maximum number of common.MaxAssetSize to accept.
// If ok == false, caller should return.
func parseAssetForm(w http.ResponseWriter, r *http.Request, maxCount uint,
) (
	board string, err error,
) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxCount)*common.MaxAssetSize)
	err = r.ParseMultipartForm(0)
	if err != nil {
		err = common.StatusError{
			Err:  err,
			Code: 400,
		}
		return
	}

	board = r.Form.Get("board")
	_, err = canPerform(w, r, board, common.BoardOwner, true)
	return
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

func setLoadingAnimation(w http.ResponseWriter, r *http.Request) {
	err := func() (err error) {
		board, err := parseAssetForm(w, r, 1)
		if err != nil {
			return
		}

		var out assets.File
		file, h, err := r.FormFile("image")
		switch err {
		case nil:
			out, err = readAssetFile(w, r, file, h, thumbnailer.Options{
				MaxSourceDims: thumbnailer.Dims{
					Width:  400,
					Height: 400,
				},
				ThumbDims: thumbnailer.Dims{
					Width:  400,
					Height: 400,
				},
				AcceptedMimeTypes: map[string]bool{
					"image/gif":  true,
					"video/webm": true,
				},
			})
			if err != nil {
				return
			}
		case http.ErrMissingFile:
			err = nil
		default:
			err = newFileError(h, err.Error())
			return
		}

		return db.SetLoadingAnimation(board, out)
	}()
	if err != nil {
		httpError(w, r, err)
	}
}

// Serve board-specific image banner files
func serveBanner(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(extractParam(r, "id"))
	if err != nil {
		text404(w)
		return
	}

	f, ok := assets.Banners.Get(extractParam(r, "board"), id)
	if !ok {
		text404(w)
		return
	}
	serveAssetFromMemory(w, r, f)
}

func serveAssetFromMemory(
	w http.ResponseWriter,
	r *http.Request,
	f assets.File,
) {
	if checkClientEtag(w, r, f.Hash) {
		return
	}

	h := w.Header()
	h.Set("ETag", f.Hash)
	h.Set("Content-Type", f.Mime)
	h.Set("Content-Length", strconv.Itoa(len(f.Data)))
	w.Write(f.Data)
}

// Serve board-specific loading animation
func serveLoadingAnimation(w http.ResponseWriter, r *http.Request) {
	serveAssetFromMemory(w, r, assets.Loading.Get(extractParam(r, "board")))
}
