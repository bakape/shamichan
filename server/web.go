/*
 Webserver
*/

package server

import (
	"bytes"
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
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(notFoundHandler)
	router.PanicHandler = panicHandler

	// Board and board JSON API pages
	router.HandlerFunc("GET", "/", redirectToDefault)
	router.HandlerFunc("GET", "/all/", allBoardHTML)
	router.HandlerFunc("GET", "/api/all/", allBoardJSON)
	for _, board := range config.Boards.Enabled {
		router.HandlerFunc("GET", "/"+board+"/", boardHTML(board))
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

	// Wrap router with extra handlers
	handler := http.Handler(router)
	if config.HTTP.TrustProxies { // Infer IP from header, if configured to
		handler = handlers.ProxyHeaders(router)
	}
	handler = handlers.CompressHandler(handler) //GZIP

	log.Println("Listening on " + config.HTTP.Addr)
	log.Fatal(http.ListenAndServe(config.HTTP.Addr, handler))
}

// Redirects to frontpage, if set, or the default board
func redirectToDefault(res http.ResponseWriter, req *http.Request) {
	if config.Frontpage != "" {
		http.ServeFile(res, req, config.Frontpage)
	} else {
		http.Redirect(res, req, "/"+config.Boards.Default+"/", 302)
	}
}

// Serves `/:board/` pages
func boardHTML(board string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		ident := lookUpIdent(req.RemoteAddr)
		if !canAccessBoard(board, ident) {
			notFound(res)
			return
		}
		isMobile := detectMobile(req)
		template := chooseTemplate(isMobile)
		etag := etagStart(boardCounter(board)) +
			htmlEtag(template.Hash, isMobile)
		if !compareEtag(res, req, ident, etag, false) {
			return
		}
		writeTemplate(res, template, ident, readBoardJSON(board, ident))
	}
}

// Serves `/api/:board/` page JSON
func boardJSON(board string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		ident := lookUpIdent(req.RemoteAddr)
		if !canAccessBoard(board, ident) {
			text404(res)
			return
		}
		if compareEtag(res, req, ident, etagStart(boardCounter(board)), true) {
			return
		}
		_, err := res.Write(readBoardJSON(board, ident))
		throw(err)
	}
}

// Reads and formats board JSON from the DB
func readBoardJSON(board string, ident Ident) []byte {
	return marshalJSON(NewReader(board, ident).GetBoard())
}

// Serves `/:board/:thread` page HTML
func threadHTML(board string) httprouter.Handle {
	return func(
		res http.ResponseWriter,
		req *http.Request,
		ps httprouter.Params,
	) {
		id, err := strconv.ParseUint(ps[0].Value, 10, 64)
		ident := lookUpIdent(req.RequestURI)
		if !validateThreadRequest(err, board, id, ident) {
			notFound(res)
			return
		}
		lastN := detectLastN(req)
		isMobile := detectMobile(req)
		template := chooseTemplate(isMobile)
		etag := etagStart(threadCounter(id)) + htmlEtag(template.Hash, isMobile)
		if lastN != 0 {
			etag += "-last" + strconv.Itoa(lastN)
		}
		if !compareEtag(res, req, ident, etag, false) {
			return
		}
		writeTemplate(
			res,
			template,
			ident,
			readThreadJSON(board, id, ident, lastN),
		)
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
		if !validateThreadRequest(err, board, id, ident) {
			text404(res)
			return
		}
		lastN := detectLastN(req)
		etag := etagStart(threadCounter(id))
		if lastN != 0 {
			etag += "-last" + strconv.Itoa(lastN)
		}
		if !compareEtag(res, req, ident, etag, true) {
			return
		}
		_, err = res.Write(readThreadJSON(board, id, ident, lastN))
		throw(err)
	}
}

// Reads and formats thread JSON from the DB
func readThreadJSON(board string, id uint64, ident Ident, lastN int) []byte {
	return marshalJSON(NewReader(board, ident).GetThread(id, lastN))
}

// Cofirm thread request is proper, thread exists and client had right of access
func validateThreadRequest(
	err error,
	board string,
	id uint64,
	ident Ident,
) bool {
	return err == nil &&
		validateOP(id, board) &&
		canAccessThread(id, board, ident)
}

// Serves HTML for the "/all/" meta-board, that contains threads from all boards
func allBoardHTML(res http.ResponseWriter, req *http.Request) {
	ident := lookUpIdent(req.RequestURI)
	if ident.Banned {
		notFound(res)
		return
	}
	isMobile := detectMobile(req)
	template := chooseTemplate(isMobile)
	etag := etagStart(postCounter()) + htmlEtag(template.Hash, isMobile)
	if !compareEtag(res, req, ident, etag, false) {
		return
	}
	writeTemplate(res, template, ident, readAllBoardJSON(ident))
}

// Serves JSON for the "/all/" meta-board, that contains threads from all boards
func allBoardJSON(res http.ResponseWriter, req *http.Request) {
	ident := lookUpIdent(req.RequestURI)
	if ident.Banned {
		text404(res)
		return
	}
	if !compareEtag(res, req, ident, etagStart(postCounter()), true) {
		return
	}
	_, err := res.Write(readAllBoardJSON(ident))
	throw(err)
}

// Reads and formats the `/all/` meta-board JSON form the DB
func readAllBoardJSON(ident Ident) []byte {
	return marshalJSON(NewReader("all", ident).GetAllBoard())
}

// Detects mobile user agents, so we can serve the apropriate template and
// client files
func detectMobile(req *http.Request) bool {
	return user_agent.New(req.UserAgent()).Mobile()
}

// Return a mobile or desktop template accordingly
func chooseTemplate(isMobile bool) templateStore {
	if isMobile {
		return resources["mobile"]
	}
	return resources["index"]
}

// Build an etag for HTML pages and check if it matches the one provided by the
// client. If yes, send 304 and return false, otherwise set headers and return
// true.
func compareEtag(
	res http.ResponseWriter,
	req *http.Request,
	ident Ident,
	etag string,
	json bool,
) bool {
	hasAuth := ident.Auth != ""
	if hasAuth {
		etag += "-" + ident.Auth
	}
	if checkClientEtag(res, req, etag) { // If etags match, no need to rerender
		return false
	}
	setHeaders(res, etag, json)
	if hasAuth { //Don't expose restricted data publicly through caches
		head := res.Header()
		head.Set("Cache-Control", head.Get("Cache-Control")+"; private")
	}
	return true
}

// Build the main part of the etag
func etagStart(counter uint64) string {
	return "W/" + idToString(counter)
}

// Extra part of the etag only used for HTML pages
func htmlEtag(hash string, isMobile bool) string {
	etag := "-" + hash
	if isMobile {
		etag += "-mobile"
	}
	return etag
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
	if val := req.Header.Get("If-None-Match"); val != "" {
		res.WriteHeader(304)
		return true
	}
	return false
}

// Concatenate post JSON with HTML template and write to client
func writeTemplate(
	res http.ResponseWriter,
	tmpl templateStore,
	ident Ident,
	data []byte,
) {
	parts := [][]byte{
		tmpl.Parts[0], data, tmpl.Parts[1],
		loginCredentials(ident), tmpl.Parts[2],
	}
	out := new(bytes.Buffer)
	for _, buf := range parts {
		_, err := out.Write(buf)
		throw(err)
	}
	_, err := res.Write(out.Bytes())
	throw(err)
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

// Inject staff login credentials, if any. These will be used to download the
// moderation JS client bundle.
func loginCredentials(ident Ident) []byte {
	// TODO: Inject the variables for our new login system

	return []byte{}
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
	board := parentBoard(id)
	thread := parentThread(id)
	ident := lookUpIdent(req.RemoteAddr)
	if board == "" || thread == 0 || !canAccessThread(thread, board, ident) {
		text404(res)
		return
	}
	data := marshalJSON(NewReader(board, ident).GetPost(id))
	etag := "W/" + hashBuffer(data)
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
