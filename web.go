/*
 Webserver
*/

package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mssola/user_agent"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func startServer() {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	router.StrictSlash(true)
	router.HandleFunc("/", redirectToDefault)

	const board = `/{board:\w+}`
	const thread = `/{thread:\d+}`

	// HTML
	router.HandleFunc("/all/", wrapHandler(false, allBoards))
	index := router.PathPrefix(board).Subrouter()
	index.HandleFunc("/", wrapHandler(false, boardPage))
	index.HandleFunc(thread, wrapHandler(false, threadPage))

	// JSON API
	api := router.PathPrefix("/api").Subrouter()
	api.NotFoundHandler = http.NotFoundHandler() // Default 404 handler for JSON
	api.HandleFunc("/config", serveConfigs)
	api.HandleFunc(`/post/{post:\d+}`, servePost)
	api.HandleFunc("/all/", wrapHandler(true, allBoards))
	posts := api.PathPrefix(board).Subrouter()
	posts.HandleFunc("/", wrapHandler(true, boardPage))
	posts.HandleFunc(thread, wrapHandler(true, threadPage))

	// Serve static assets
	if config.Hard.HTTP.ServeStatic {
		router.PathPrefix("/").Handler(http.FileServer(http.Dir("./www")))
	}

	var handler http.Handler = router

	if config.Hard.HTTP.TrustProxies { // Infer IP from header, if configured to
		handler = handlers.ProxyHeaders(router)
	}
	if config.Hard.HTTP.Gzip {
		handler = handlers.CompressHandler(handler)
	}
	handler = getIdent(handler)
	handler = handlers.RecoveryHandler( // Return status 500 on goroutine panic
		handlers.PrintRecoveryStack(true),
	)(handler)

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

type handlerFunction func(http.ResponseWriter, *http.Request)
type handlerWrapper func(bool, http.ResponseWriter, *http.Request)

// wrapHandler returns a function with the first bool argument already assigned
func wrapHandler(json bool, handler handlerWrapper) handlerFunction {
	return func(res http.ResponseWriter, req *http.Request) {
		handler(json, res, req)
	}
}

// Handles `/board/` page requests
func boardPage(jsonOnly bool, res http.ResponseWriter, req *http.Request) {
	in := indexPage{res: res, req: req, json: jsonOnly}
	board := mux.Vars(req)["board"]

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

// Handles `/board/thread` requests
func threadPage(jsonOnly bool, res http.ResponseWriter, req *http.Request) {
	in := indexPage{res: res, req: req, json: jsonOnly}
	vars := mux.Vars(req)
	board := vars["board"]
	id, err := strconv.ParseUint(vars["thread"], 10, 64)
	throw(err)
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

// Handles the "all" meta-board, that contains threads from all boards
func allBoards(jsonOnly bool, res http.ResponseWriter, req *http.Request) {
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

// Stores common variables and methods for both board and thread pages
type indexPage struct {
	res         http.ResponseWriter
	req         *http.Request
	validate    func() bool
	getCounter  func() uint64 // Progress counter used for building etags
	getPostData func() []byte // Post model JSON data
	lastN       int
	json        bool // Serve HTML from template or just JSON
	isMobile    bool
	template    templateStore
	ident       Ident
}

// Shared logic for handling both board and thread pages
func (in *indexPage) process(board string) {
	in.ident = extractIdent(in.res, in.req)
	if !in.validate() {
		in.res.WriteHeader(404)
		if !in.json {
			custom404(in.res)
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
			in.res.Write(postData)
			return
		}

		// Concatenate post JSON with template and write to client
		parts := in.template.Parts
		html := new(bytes.Buffer)
		html.Write(parts[0])
		html.Write(in.getPostData())
		html.Write(parts[1])
		html.Write(loginCredentials(in.ident))
		html.Write(parts[2])
		in.res.Write(html.Bytes())
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
	if ifNoneMatch, ok := req.Header["If-None-Match"]; ok {
		for _, clientEtag := range ifNoneMatch {
			if clientEtag == etag {
				res.WriteHeader(304)
				return true
			}
		}
	}
	return false
}

// Read client Identity struct, which was attached to the requests further
// upstream
func extractIdent(res http.ResponseWriter, req *http.Request) Ident {
	ident, ok := context.Get(req, "ident").(Ident)
	if !ok {
		res.WriteHeader(500)
		throw(errors.New("Failed Ident type assertion"))
	}
	return ident
}

// Custom error page for requests that don't match a router
func notFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(404)
	custom404(res)
}

// Serve the custom error page
func custom404(res http.ResponseWriter) {
	copyFile("www/404.html", res)
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
	parsed, err := url.ParseRequestURI(req.RequestURI)
	throw(err)
	lastNSlice, ok := parsed.Query()["lastN"]
	if ok && len(lastNSlice) > 0 {
		lastN, err := strconv.Atoi(lastNSlice[0])
		throw(err)
		if lastN <= 500 {
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
	res.Write(marshalJSON(clientConfig))
}

// Serve a single post as JSON
func servePost(res http.ResponseWriter, req *http.Request) {
	id, err := strconv.ParseUint(mux.Vars(req)["post"], 10, 64)
	throw(err)
	board := parentBoard(id)
	thread := parentThread(id)
	ident := extractIdent(res, req)
	if board == "" || thread == 0 || !canAccessThread(thread, board, ident) {
		res.WriteHeader(404)
		return
	}
	data := marshalJSON(NewReader(board, ident).GetPost(id))
	etag := "W/" + hashBuffer(data)
	if checkClientEtags(res, req, etag) {
		return
	}
	setHeaders(res, etag, true)
	res.Write(data)
}
