/*
 Webserver
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	r "github.com/dancannon/gorethink"
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mssola/user_agent"
	"log"
	"net/http"
	"strconv"
)

func startServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", redirectToDefault)
	router.HandleFunc(`/{board:\w+}`, addTrailingSlash)
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	sub := router.Path(`/{board:\w+}/`).Subrouter()
	sub.HandleFunc("/", boardPage)

	// Serve static assets
	if config.Hard.HTTP.ServeStatic {
		// TODO: Apply headers, depending on debug mode
		router.PathPrefix("/").Handler(http.FileServer(http.Dir("./www")))
	}

	// Infer IP from header, if configured to
	var handler http.Handler
	if config.Hard.HTTP.TrustProxies {
		handler = handlers.ProxyHeaders(router)
	} else {
		handler = router
	}
	handler = getIdent(handler)

	log.Println("Listening on " + config.Hard.HTTP.Addr)
	http.ListenAndServe(config.Hard.HTTP.Addr, handler)
}

// Attach client access rights to request
func getIdent(handler http.Handler) http.Handler {
	fn := func(res http.ResponseWriter, req *http.Request) {
		context.Set(req, "ident", lookUpIdent(req.RemoteAddr))

		// Call the next handler in the chain
		handler.ServeHTTP(res, req)
	}

	return http.HandlerFunc(fn)
}

// Redirects to frontpage, if set, or the default board
func redirectToDefault(res http.ResponseWriter, req *http.Request) {
	if config.Frontpage != "" {
		http.ServeFile(res, req, config.Frontpage)
	} else {
		http.Redirect(res, req, "/"+config.Boards.Default+"/", 302)
	}
}

// Redirects `/board` to `/board/`. The client parses the URL to determine what
// page it is on. So we need the trailing slash for easier board determination
// and consistency.
func addTrailingSlash(res http.ResponseWriter, req *http.Request) {
	http.Redirect(res, req, "/"+mux.Vars(req)["board"]+"/", 301)
}

// handles `/board/` page requests
func boardPage(res http.ResponseWriter, req *http.Request) {
	board := mux.Vars(req)["board"]
	ident, ok := context.Get(req, "ident").(Ident)
	if !ok {
		throw(errors.New("Failed Ident type assertion"))
		res.WriteHeader(500)
	}
	if !canAccessBoard(board, ident) {
		send404(res)
	}
	var counter int
	rGet(r.Table("main").Get("histCounts").
		Field("board").
		Default(0),
	).
		One(counter)
	needRender, isMobile, isRetarded := validateEtag(res, req, counter, ident)
	if needRender {
		raw := Newreader(board, ident).GetBoard()
		postData, err := json.Marshal(raw)
		throw(err)
		var template [][]byte
		if !isMobile {
			template = resources["index"].Parts
		} else {
			template = resources["mobile"].Parts
		}
		res.Write(template[0])
		res.Write([]byte(strconv.FormatBool(isRetarded)))
		res.Write(template[1])
		res.Write(postData)
		res.Write(loginCredentials(ident))
		res.Write(template[2])
	}
}

func notFoundHandler(res http.ResponseWriter, req *http.Request) {
	send404(res)
}

func send404(res http.ResponseWriter) {
	res.WriteHeader(404)
	copyFile("www/404.html", res)
}

// Build an etag and check if it mathces the one provided by the client. If yes,
// send 304 and return false, otherwise set headers and return true.
func validateEtag(res http.ResponseWriter,
	req *http.Request,
	counter int,
	ident Ident,
) (needRender, isMobile, isRetarded bool) {
	var etag string
	etag, isMobile, isRetarded = buildEtag(req, counter)
	needRender = true
	if config.Hard.Debug {
		setHeaders(res, noCacheHeaders)
		return
	}
	hasAuth := ident.Auth != ""
	if hasAuth {
		etag += "-" + ident.Auth
	}

	// Etags match. No need to rerender.
	if ifNoneMatch, ok := req.Header["If-None-Match"]; ok {
		for _, clientEtag := range ifNoneMatch {
			if clientEtag == etag {
				res.WriteHeader(304)
				needRender = false
				return
			}
		}
	}

	setHeaders(res, vanillaHeaders)
	res.Header().Set("ETag", etag)
	if hasAuth {
		res.Header().Add("Cache-Control", ", private")
	}
	return
}

// Build the main part of the etag
func buildEtag(req *http.Request, counter int) (etag string,
	isMobile, isRetarded bool,
) {
	ua := user_agent.New(req.UserAgent())
	isMobile = ua.Mobile()
	context.Set(req, "isMobile", isMobile)

	browser, _ := ua.Browser()
	isRetarded = true
	supported := [...]string{"Chrome", "Chromium", "Opera", "Firefox"}
	for _, supportedBrowser := range supported {
		if browser == supportedBrowser {
			isRetarded = false
			break
		}
	}
	context.Set(req, "isRetarded", isRetarded)

	var hash string
	if !isMobile {
		hash = resources["index"].Hash
	} else {
		hash = resources["mobile"].Hash
	}

	etag = fmt.Sprintf(`W/%v-%v`, counter, hash)
	if isMobile {
		etag += "-mobile"
	}
	if isRetarded {
		etag += "-retarded"
	}
	return
}

var noCacheHeaders = stringMap{
	"X-Frame-Options": "sameorigin",
	"Expires":         "Thu, 01 Jan 1970 00:00:00 GMT",
	"Cache-Control":   "no-cache, no-store",
}
var vanillaHeaders = stringMap{
	"Content-Type":    "text/html; charset=UTF-8",
	"X-Frame-Options": "sameorigin",
	"Cache-Control":   "max-age=0, must-revalidate",
	"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
}

func setHeaders(res http.ResponseWriter, headers stringMap) {
	for key, val := range headers {
		res.Header().Set(key, val)
	}
}

// Inject staff login credentials, if any. These will be used to download the
// moderation JS client bundle.
func loginCredentials(ident Ident) []byte {
	// TODO: Inject the variables for our new login system

	return []byte{}
}
