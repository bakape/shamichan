package server

import (
	"testing"

	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/types"
)

func TestServeIndexTemplate(t *testing.T) {
	desktop := templates.Store{
		HTML: []byte("desktop"),
		Hash: "dhash",
	}
	templates.Set("index", desktop)
	headers := map[string]string{
		"Content-Type": "text/html",
	}
	setBoards(t, "a")

	t.Run("initial", func(t *testing.T) {
		t.Parallel()

		rec, req := newPair("/a/")
		router.ServeHTTP(rec, req)
		assertBody(t, rec, string(desktop.HTML))
		assertEtag(t, rec, desktop.Hash)
		assertHeaders(t, rec, headers)
	})

	t.Run("etag matches", func(t *testing.T) {
		t.Parallel()

		rec, req := newPair("/a/")
		req.Header.Set("If-None-Match", desktop.Hash)
		router.ServeHTTP(rec, req)
		assertCode(t, rec, 304)
	})
}

func TestThreadHTML(t *testing.T) {
	assertTableClear(t, "threads")
	assertInsert(t, "threads", types.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	setBoards(t, "a")
	body := []byte("body")
	templates.Set("index", templates.Store{
		HTML: body,
		Hash: "hash",
	})

	t.Run("unparsable thread number", func(t *testing.T) {
		t.Parallel()

		rec, req := newPair("/a/www")
		router.ServeHTTP(rec, req)
		assertCode(t, rec, 400)
	})
	t.Run("nonexistent thread", func(t *testing.T) {
		t.Parallel()

		rec, req := newPair("/a/22")
		router.ServeHTTP(rec, req)
		assertCode(t, rec, 404)
	})
	t.Run("thread exists", func(t *testing.T) {
		t.Parallel()

		rec, req := newPair("/a/1")
		router.ServeHTTP(rec, req)
		assertBody(t, rec, string(body))
	})
}
