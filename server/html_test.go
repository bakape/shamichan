package server

import (
	"github.com/bakape/meguca/templates"
	. "gopkg.in/check.v1"
)

func (w *WebServer) TestServeIndexTemplate(c *C) {
	const (
		desktopUA = "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 " +
			"(KHTML, like Gecko) Chrome/41.0.2228.0 Safari/537.36"
		mobileUA = "Mozilla/5.0 (Linux; Android 4.1.1; Galaxy Nexus" +
			" Build/JRO03C) AppleWebKit/535.19 (KHTML, like Gecko)" +
			" Chrome/18.0.1025.166 Mobile Safari/535.19"
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

	// Desktop
	rec, req := newPair(c, "/a/")
	req.Header.Set("User-Agent", desktopUA)
	w.r.ServeHTTP(rec, req)
	assertBody(rec, string(desktop.HTML), c)
	assertEtag(rec, desktop.Hash, c)
	assertHeaders(c, rec, headers)

	// Mobile
	rec, req = newPair(c, "/a/")
	req.Header.Set("User-Agent", mobileUA)
	w.r.ServeHTTP(rec, req)
	assertBody(rec, string(mobile.HTML), c)
	assertEtag(rec, mobile.Hash+"-mobile", c)
	assertHeaders(c, rec, headers)

	// Etag matches
	rec, req = newPair(c, "/a/")
	req.Header.Set("If-None-Match", desktop.Hash)
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestThreadHTML(c *C) {
	setupPosts(c)
	body := []byte("body")
	templates.Set("index", templates.Store{
		HTML: body,
		Hash: "hash",
	})
	webRoot = "test"

	// Unparsable thread number
	rec, req := newPair(c, "/a/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Non-existant thread
	rec, req = newPair(c, "/a/22")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Thread exists
	rec, req = newPair(c, "/a/1")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, string(body), c)
}
