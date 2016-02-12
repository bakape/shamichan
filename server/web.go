/*
 Webserver
*/

package server

import (
	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// Used for overriding during tests
var (
	webRoot      = "./www"
	imageWebRoot = "./img"
)

func startWebServer() {
	log.Println("Listening on " + config.HTTP.Addr)
	log.Fatal(http.ListenAndServe(config.HTTP.Addr, wrapRouter(createRouter())))
}

// Create the monolithic router for routing HTTP requests. Separated into own
// function for easier testability.
func createRouter() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(notFoundHandler)
	router.PanicHandler = panicHandler

	// Board and board JSON API pages
	router.HandlerFunc("GET", "/", redirectToDefault)
	router.HandlerFunc("GET", "/all/", serveIndexTemplate)
	router.HandlerFunc("GET", "/api/all/", allBoardJSON)
	for _, board := range config.Boards.Enabled {
		router.HandlerFunc("GET", "/"+board+"/", serveIndexTemplate)
		router.HandlerFunc("GET", "/api/"+board+"/", boardJSON(board))

		// Thread pages
		router.GET("/"+board+"/:thread", threadHTML(board))
		router.GET("/api/"+board+"/:thread", threadJSON(board))
	}

	// Other JSON API handlers
	router.HandlerFunc("GET", "/api/config", serveConfigs)
	router.GET("/api/post/:post", servePost)

	// Static assets
	router.ServeFiles("/ass/*filepath", http.Dir("./www"))
	router.GET("/img/*filepath", serveImages)

	// Image upload
	router.HandlerFunc("POST", "/upload", NewImageUpload)

	return router
}

// Wraps the router in additional helper handlers. Seperated for easier
// testability.
func wrapRouter(router *httprouter.Router) http.Handler {
	// Wrap router with extra handlers
	handler := http.Handler(router)
	if config.HTTP.TrustProxies { // Infer IP from header, if configured to
		handler = handlers.ProxyHeaders(router)
	}
	handler = handlers.CompressHandler(handler) //GZIP
	return handler
}

// Redirects to frontpage, if set, or the default board
func redirectToDefault(res http.ResponseWriter, req *http.Request) {
	if config.Frontpage != "" {
		http.ServeFile(res, req, config.Frontpage)
	} else {
		http.Redirect(res, req, "/"+config.Boards.Default+"/", 302)
	}
}

func serveIndexTemplate(res http.ResponseWriter, req *http.Request) {
	isMobile := user_agent.New(req.UserAgent()).Mobile()
	var template templateStore
	if isMobile {
		template = resources["mobile"]
	} else {
		template = resources["index"]
	}
	etag := template.Hash
	if isMobile {
		etag += "-mobile"
	}
	if !compareEtag(res, req, etag, false) {
		return
	}
	_, err := res.Write(template.HTML)
	throw(err)
}

// Serves `/api/:board/` page JSON
func boardJSON(board string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if !compareEtag(res, req, etagStart(boardCounter(board)), true) {
			return
		}
		ident := lookUpIdent(req.RemoteAddr)
		_, err := res.Write(marshalJSON(NewReader(board, ident).GetBoard()))
		throw(err)
	}
}

// Serves `/:board/:thread` page HTML
func threadHTML(board string) httprouter.Handle {
	return func(
		res http.ResponseWriter,
		req *http.Request,
		ps httprouter.Params,
	) {
		id, err := strconv.ParseUint(ps[0].Value, 10, 64)
		if !validateThreadRequest(err, board, id) {
			notFound(res)
			return
		}
		serveIndexTemplate(res, req)
	}
}

// Serves `/api/:board/:thread` page JSON
func threadJSON(board string) httprouter.Handle {
	return func(
		res http.ResponseWriter,
		req *http.Request,
		ps httprouter.Params,
	) {
		id, err := strconv.ParseUint(ps[0].Value, 10, 64)
		ident := lookUpIdent(req.RequestURI)
		if !validateThreadRequest(err, board, id) {
			text404(res)
			return
		}
		if !compareEtag(res, req, etagStart(threadCounter(id)), true) {
			return
		}
		data := marshalJSON(
			NewReader(board, ident).GetThread(id, detectLastN(req)),
		)
		_, err = res.Write(data)
		throw(err)
	}
}

// Cofirm thread request is proper, thread exists and client had right of access
func validateThreadRequest(err error, board string, id uint64) bool {
	return err == nil && validateOP(id, board)
}

// Serves JSON for the "/all/" meta-board, that contains threads from all boards
func allBoardJSON(res http.ResponseWriter, req *http.Request) {
	if !compareEtag(res, req, etagStart(postCounter()), true) {
		return
	}
	ident := lookUpIdent(req.RemoteAddr)
	_, err := res.Write(marshalJSON(NewReader("all", ident).GetAllBoard()))
	throw(err)
}

// Build an etag for HTML pages and check if it matches the one provided by the
// client. If yes, send 304 and return false, otherwise set headers and return
// true.
func compareEtag(
	res http.ResponseWriter,
	req *http.Request,
	etag string,
	json bool,
) bool {
	if checkClientEtag(res, req, etag) { // If etags match, no need to rerender
		return false
	}
	setHeaders(res, etag, json)
	return true
}

// Build the main part of the etag
func etagStart(counter uint64) string {
	return "W/" + idToString(counter)
}

/*
 Check is any of the etags the client provides in the "If-None-Match" header
 match the generated etag. If yes, write 304 and return true.
*/
func checkClientEtag(
	res http.ResponseWriter,
	req *http.Request,
	etag string,
) bool {
	if etag == req.Header.Get("If-None-Match") {
		res.WriteHeader(304)
		return true
	}
	return false
}

// Serve custom error page
func notFound(res http.ResponseWriter) {
	setErrorHeaders(res)
	res.WriteHeader(404)
	copyFile(webRoot+"/404.html", res)
}

// Addapter for using notFound as a route handler
func notFoundHandler(res http.ResponseWriter, _ *http.Request) {
	notFound(res)
}

// Text-only 404 response
func text404(res http.ResponseWriter) {
	http.Error(res, "404 Not found", 404)
}

// Set HTTP headers for returning custom error pages
func setErrorHeaders(res http.ResponseWriter) {
	res.Header().Set("Content-Type", "text/html; charset=UTF-8")
	res.Header().Set("X-Content-Type-Options", "nosniff")
}

// Serve server error page and log stack trace on error
func panicHandler(res http.ResponseWriter, req *http.Request, err interface{}) {
	setErrorHeaders(res)
	res.WriteHeader(500)
	copyFile(webRoot+"/50x.html", res)
	logError(req, err)
}

// Set HTTP headers to the response object
func setHeaders(res http.ResponseWriter, etag string, json bool) {
	vanillaHeaders := map[string]string{
		"X-Frame-Options": "sameorigin",
		"Cache-Control":   "max-age=0, must-revalidate",
		"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
	}
	head := res.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}

	head.Set("ETag", etag)

	var contentType string
	if json {
		contentType = "application/json"
	} else {
		contentType = "text/html"
	}
	head.Set("Content-Type", contentType+"; charset=UTF-8")
}

// Validate the client's last N posts to display setting
func detectLastN(req *http.Request) int {
	query := req.URL.Query().Get("lastN")
	if query != "" {
		lastN, err := strconv.Atoi(query)
		if err == nil && lastN <= 500 {
			return lastN
		}
	}
	return 0
}

// Serve public configuration information as JSON
func serveConfigs(res http.ResponseWriter, req *http.Request) {
	etag := "W/" + configHash
	if checkClientEtag(res, req, etag) {
		return
	}
	setHeaders(res, etag, true)
	_, err := res.Write(marshalJSON(clientConfig))
	throw(err)
}

// Serve a single post as JSON
func servePost(
	res http.ResponseWriter,
	req *http.Request,
	ps httprouter.Params,
) {
	id, err := strconv.ParseUint(ps[0].Value, 10, 64)
	if err != nil {
		text404(res)
		return
	}
	post := NewReader("", lookUpIdent(req.RemoteAddr)).GetPost(id)
	if post.ID == 0 { // No post in the database or no access
		text404(res)
		return
	}
	data := marshalJSON(post)
	etag := hashBuffer(data)
	if checkClientEtag(res, req, etag) {
		return
	}
	setHeaders(res, etag, true)
	_, err = res.Write(data)
	throw(err)
}

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set seperate caching policies for them.
func serveImages(
	res http.ResponseWriter,
	req *http.Request,
	ps httprouter.Params,
) {
	file, err := os.Open(imageWebRoot + httprouter.CleanPath(ps[0].Value))
	defer file.Close()
	if err != nil {
		text404(res)
		return
	}
	headers := res.Header()

	// Fake etag to stop agressive browser cache busting
	if checkClientEtag(res, req, "0") {
		return
	}
	headers.Set("ETag", "0")

	// max-age set to 350 days. Some caches and browsers ignore max-age, if it
	// is a year or greater, so keep it a little bellow.
	headers.Set("Cache-Control", "max-age=30240000")
	headers.Set("X-Frame-Options", "sameorigin")
	_, err = io.Copy(res, file)
	throw(err)
}
