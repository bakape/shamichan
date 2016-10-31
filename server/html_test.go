package server

import (
	"testing"

	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/types"
)

func TestServeIndexTemplate(t *testing.T) {
	const (
		desktopUA = "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"
		mobileUA  = "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.166 Mobile Safari/535.19"
	)
	desktop := templates.Store{
		HTML: []byte("desktop"),
		Hash: "dhash",
	}
	mobile := templates.Store{
		HTML: []byte("mobile"),
		Hash: "mhash",
	}
	templates.Set("index", desktop)
	templates.Set("mobile", mobile)
	headers := map[string]string{
		"Content-Type": "text/html",
	}
	setBoards(t, "a")

	t.Run("desktop", func(t *testing.T) {
		t.Parallel()

		rec, req := newPair("/a/")
		req.Header.Set("User-Agent", desktopUA)
		router.ServeHTTP(rec, req)
		assertBody(t, rec, string(desktop.HTML))
		assertEtag(t, rec, desktop.Hash)
		assertHeaders(t, rec, headers)
	})

	t.Run("mobile", func(t *testing.T) {
		t.Parallel()

		rec, req := newPair("/a/")
		req.Header.Set("User-Agent", mobileUA)
		router.ServeHTTP(rec, req)
		assertBody(t, rec, string(mobile.HTML))
		assertEtag(t, rec, mobile.Hash+"-mobile")
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
		assertCode(t, rec, 404)
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
