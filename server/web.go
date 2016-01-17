/*
 Webserver
*/

package server

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

func startWebServer() {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(notFound)
	router.PanicHandler = panicHandler

	// Board and board JSON API pages
	router.HandlerFunc("GET", "/", redirectToDefault)
	router.HandlerFunc("GET", "/all/", allBoards(false))
	router.HandlerFunc("GET", "/api/all/", allBoards(true))
	for _, board := range config.Boards.Enabled {
		router.HandlerFunc("GET", "/"+board+"/", boardPage(false, board))
		router.HandlerFunc("GET", "/api/"+board+"/", boardPage(true, board))

		// Thread pages
		router.GET("/"+board+"/:thread", threadPage(false, board))
		router.GET("/api/"+board+"/:thread", threadPage(true, board))
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
	if config.Hard.HTTP.TrustProxies { // Infer IP from header, if configured to
		handler = handlers.ProxyHeaders(router)
	}
	handler = handlers.CompressHandler(handler) //GZIP

	log.Println("Listening on " + config.Hard.HTTP.Addr)
	log.Fatal(http.ListenAndServe(config.Hard.HTTP.Addr, handler))
}

// Redirects to frontpage, if set, or the default board
func redirectToDefault(res http.ResponseWriter, req *http.Request) {
	if config.Frontpage != "" {
		http.ServeFile(res, req, config.Frontpage)
	} else {
		http.Redirect(res, req, "/"+config.Boards.Default+"/", 302)
	}
}

// boardPage constructs a handler for `/board/` pages for serving either HTML
// or JSON
func boardPage(jsonOnly bool, board string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		in := indexPage{res: res, req: req, json: jsonOnly}

		in.validate = func() bool {
			return canAccessBoard(board, in.ident)
		}

		in.getCounter = func() uint64 {
			return boardCounter(board)
		}

		in.getPostData = func() []byte {
			return marshalJSON(NewReader(board, in.ident).GetBoard())
		}

		in.process(board)
	}
}

// Same as above for `/board/thread` pages
func threadPage(jsonOnly bool, board string) httprouter.Handle {
	return func(
		res http.ResponseWriter,
		req *http.Request,
		ps httprouter.Params,
	) {
		id, err := strconv.ParseUint(ps[0].Value, 10, 64)
		if err != nil {
			if jsonOnly {
				text404(res)
			} else {
				notFound(res, req)
			}
			return
		}
		in := indexPage{res: res, req: req, json: jsonOnly}
		in.lastN = detectLastN(req)

		in.validate = func() bool {
			return validateOP(id, board) && canAccessThread(id, board, in.ident)
		}

		in.getCounter = func() uint64 {
			return threadCounter(id)
		}

		in.getPostData = func() []byte {
			return marshalJSON(NewReader(board, in.ident).GetThread(id, in.lastN))
		}

		in.process(board)
	}
}

// Same as above for the "/all/" meta-board, that contains threads from all
// boards
func allBoards(jsonOnly bool) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		in := indexPage{res: res, req: req, json: jsonOnly}

		in.validate = func() bool {
			return !in.ident.Banned
		}

		in.getCounter = postCounter

		in.getPostData = func() []byte {
			return marshalJSON(NewReader("all", in.ident).GetAllBoard())
		}

		in.process("all")
	}
}

// Stores common variables and methods for both board and thread pages
type indexPage struct {
	json        bool // Serve only JSON, not HTML
	isMobile    bool
	ident       Ident
	lastN       int
	res         http.ResponseWriter
	req         *http.Request
	template    templateStore
	validate    func() bool
	getCounter  func() uint64 // Progress counter used for building etags
	getPostData func() []byte // Post model JSON data
}

// Shared logic for handling both board and thread pages
func (in *indexPage) process(board string) {
	in.ident = lookUpIdent(in.req.RemoteAddr)
	if !in.validate() {
		if in.json {
			text404(in.res)
		} else {
			notFound(in.res, in.req)
		}
		return
	}
	in.isMobile = user_agent.New(in.req.UserAgent()).Mobile()

	// Choose template to use
	if !in.json {
		if !in.isMobile {
			in.template = resources["index"]
		} else {
			in.template = resources["mobile"]
		}
	}
	if in.validateEtag() {
		postData := in.getPostData()
		if in.json { //Only the JSON
			_, err := in.res.Write(postData)
			throw(err)
			return
		}

		// Concatenate post JSON with template and write to client
		parts := in.template.Parts
		_, err := in.res.Write(concatBuffers(
			parts[0], in.getPostData(), parts[1], loginCredentials(in.ident),
			parts[2],
		))
		throw(err)
	}
}

// Build an etag and check if it matches the one provided by the client. If yes,
// send 304 and return false, otherwise set headers and return true.
func (in *indexPage) validateEtag() bool {
	etag := in.buildEtag()
	hasAuth := in.ident.Auth != ""
	if hasAuth {
		etag += "-" + in.ident.Auth
	}
	if in.lastN != 0 {
		etag += fmt.Sprintf("-last%v", in.lastN)
	}

	// If etags match, no need to rerender
	if checkClientEtags(in.res, in.req, etag) {
		return false
	}

	setHeaders(in.res, etag, in.json)
	if hasAuth {
		in.res.Header().Add("Cache-Control", ", private")
	}
	return true
}

// Build the main part of the etag
func (in *indexPage) buildEtag() string {
	etag := "W/" + idToString(in.getCounter())
	if !in.json {
		etag += "-" + in.template.Hash
		if in.isMobile {
			etag += "-mobile"
		}
	}
	return etag
}

/*
 Check is any of the etags the client provides in the "If-None-Match" header
 match the generated etag. If yes, write 304 and return true.
*/
func checkClientEtags(
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

// Custom error page for requests that don't match a router
func notFound(res http.ResponseWriter, req *http.Request) {
	setErrorHeaders(res)
	res.WriteHeader(404)
	copyFile("www/404.html", res)
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
	copyFile("./www/50x.html", res)
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
	if checkClientEtags(res, req, etag) {
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
	if checkClientEtags(res, req, etag) {
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
	file, err := os.Open("./img/" + httprouter.CleanPath(ps[0].Value))
	if err != nil {
		if os.IsNotExist(err) {
			text404(res)
			return
		}
		panic(err)
	}
	defer file.Close()
	headers := res.Header()

	// Fake etag, to stop agressive browser cache busting
	if checkClientEtags(res, req, "0") {
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
