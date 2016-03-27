/*
 Webserver
*/

package server

import (
	"encoding/json"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/server/websockets"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"
	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// Used for overriding during tests
var (
	webRoot      = "www"
	imageWebRoot = "img"
)

func startWebServer() (err error) {
	log.Println("Listening on " + config.Config.HTTP.Addr)
	conf := config.Config.HTTP
	r := createRouter()
	if conf.SSL {
		err = http.ListenAndServeTLS(conf.Addr, conf.Cert, conf.Key, r)
	} else {
		err = http.ListenAndServe(conf.Addr, r)
	}
	if err != nil {
		return util.WrapError("Error starting web server", err)
	}
	return
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
	for _, board := range config.Config.Boards.Enabled {
		router.HandlerFunc("GET", "/"+board+"/", serveIndexTemplate)
		router.HandlerFunc("GET", "/api/"+board+"/", boardJSON(board))

		// Thread pages
		router.GET("/"+board+"/:thread", threadHTML(board))
		router.GET("/api/"+board+"/:thread", threadJSON(board))
	}

	// Other JSON API handlers
	router.HandlerFunc("GET", "/api/config", serveConfigs)
	router.GET("/api/post/:post", servePost)

	// Websocket API
	router.HandlerFunc("GET", "/socket", websockets.Handler)

	// Static assets
	router.ServeFiles("/ass/*filepath", http.Dir("./www"))
	router.GET("/img/*filepath", serveImages)

	// Image upload
	router.HandlerFunc("POST", "/upload", imager.NewImageUpload)

	return router
}

// Wraps the router in additional helper handlers. Seperated for easier
// testability.
func wrapRouter(router *httprouter.Router) http.Handler {
	// Wrap router with extra handlers
	handler := http.Handler(router)
	if config.Config.HTTP.TrustProxies { // Infer IP from header, if configured to
		handler = handlers.ProxyHeaders(router)
	}
	handler = handlers.CompressHandler(handler) //GZIP
	return handler
}

// Redirects to frontpage, if set, or the default board
func redirectToDefault(res http.ResponseWriter, req *http.Request) {
	if config.Config.Frontpage != "" {
		http.ServeFile(res, req, config.Config.Frontpage)
	} else {
		http.Redirect(res, req, "/"+config.Config.Boards.Default+"/", 302)
	}
}

func serveIndexTemplate(res http.ResponseWriter, req *http.Request) {
	isMobile := user_agent.New(req.UserAgent()).Mobile()
	var template templates.Store
	if isMobile {
		template = templates.Resources["mobile"]
	} else {
		template = templates.Resources["index"]
	}
	etag := template.Hash
	if isMobile {
		etag += "-mobile"
	}
	if !compareEtag(res, req, etag, false) {
		return
	}
	_, err := res.Write(template.HTML)
	if err != nil {
		panicHandler(res, req, err)
		return
	}
}

// Serves `/api/:board/` page JSON
func boardJSON(board string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		counter, err := db.BoardCounter(board)
		if err != nil {
			panicHandler(res, req, err)
			return
		}
		if !compareEtag(res, req, etagStart(counter), true) {
			return
		}
		ident := auth.LookUpIdent(req.RemoteAddr)
		writeJSON(res, req, func() (interface{}, error) {
			return db.NewReader(board, ident).GetBoard()
		})
	}
}

// writeJSON is a helper function for calling the same error handler on the
// first to fail operation, when writing JSON to the client
func writeJSON(
	res http.ResponseWriter,
	req *http.Request,
	source func() (interface{}, error),
) {
	var (
		data interface{}
		JSON []byte
	)
	fns := []func() error{
		func() (err error) {
			data, err = source()
			return
		},
		func() (err error) {
			JSON, err = json.Marshal(data)
			return
		},
		func() (err error) {
			_, err = res.Write(JSON)
			return
		},
	}
	if err := util.Waterfall(fns); err != nil {
		panicHandler(res, req, err)
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
		valid, err := validateThreadRequest(err, board, id)
		if err != nil {
			panicHandler(res, req, err)
			return
		}
		if !valid {
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
		valid, err := validateThreadRequest(err, board, id)
		if err != nil {
			panicHandler(res, req, err)
			return
		}
		if !valid {
			text404(res)
			return
		}

		ident := auth.LookUpIdent(req.RequestURI)

		counter, err := db.ThreadCounter(id)
		if err != nil {
			panicHandler(res, req, err)
			return
		}
		if !compareEtag(res, req, etagStart(counter), true) {
			return
		}

		writeJSON(res, req, func() (interface{}, error) {
			return db.NewReader(board, ident).GetThread(id, detectLastN(req))
		})
	}
}

// Cofirm thread request is proper, thread exists and client had right of access
func validateThreadRequest(err error, board string, id uint64) (bool, error) {
	if err != nil {
		return false, nil
	}
	valid, err := db.ValidateOP(id, board)
	if err != nil {
		return false, err
	}
	return valid, nil
}

// Serves JSON for the "/all/" meta-board, that contains threads from all boards
func allBoardJSON(res http.ResponseWriter, req *http.Request) {
	counter, err := db.PostCounter()
	if err != nil {
		panicHandler(res, req, err)
		return
	}
	if !compareEtag(res, req, etagStart(counter), true) {
		return
	}
	ident := auth.LookUpIdent(req.RemoteAddr)
	writeJSON(res, req, func() (interface{}, error) {
		return db.NewReader("all", ident).GetAllBoard()
	})
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
	// If etags match, no need to rerender
	if checkClientEtag(res, req, etag) {
		return false
	}
	setHeaders(res, etag, json)
	return true
}

// Build the main part of the etag
func etagStart(counter uint64) string {
	return "W/" + util.IDToString(counter)
}

// Check is any of the etags the client provides in the "If-None-Match" header
// match the generated etag. If yes, write 304 and return true.
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
	err := util.CopyFile(filepath.FromSlash(webRoot+"/404.html"), res)
	if err != nil {
		log.Panicln(err)
	}
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
	copyErr := util.CopyFile(filepath.FromSlash(webRoot+"/50x.html"), res)
	if copyErr != nil {
		log.Println(copyErr)
	}
	util.LogError(req, err)
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
	etag := "W/" + config.Hash
	if checkClientEtag(res, req, etag) {
		return
	}
	setHeaders(res, etag, true)
	_, err := res.Write(config.ClientConfig)
	if err != nil {
		panicHandler(res, req, err)
	}
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

	post, err := db.NewReader("", auth.LookUpIdent(req.RemoteAddr)).GetPost(id)
	if err != nil {
		panicHandler(res, req, err)
		return
	}
	if post.ID == 0 { // No post in the database or no access
		text404(res)
		return
	}

	data, err := json.Marshal(post)
	if err != nil {
		panicHandler(res, req, err)
		return
	}

	etag, err := util.HashBuffer(data)
	if err != nil {
		panicHandler(res, req, err)
		return
	}
	if checkClientEtag(res, req, etag) {
		return
	}

	setHeaders(res, etag, true)
	_, err = res.Write(data)
	if err != nil {
		panicHandler(res, req, err)
		return
	}
}

// More performant handler for serving image assets. These are immutable
// (except deletion), so we can also set seperate caching policies for them.
func serveImages(
	res http.ResponseWriter,
	req *http.Request,
	ps httprouter.Params,
) {
	file, err := os.Open(
		filepath.FromSlash(imageWebRoot + httprouter.CleanPath(ps[0].Value)),
	)
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
	if err != nil {
		panicHandler(res, req, err)
		return
	}
}
