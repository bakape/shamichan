/*
 Webserver
*/

package main

import (
	"bytes"
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

	index := router.PathPrefix(board).Subrouter()
	index.HandleFunc("/", wrapHandler(false, boardPage))
	index.HandleFunc(thread, wrapHandler(false, threadPage))

	api := router.PathPrefix("/api").Subrouter()
	api.NotFoundHandler = http.NotFoundHandler() // Default 404 handler for JSON
	api.HandleFunc("/config", serveConfigs)
	api.HandleFunc(`/post/{post:\d+}`, servePost)
	posts := api.PathPrefix(board).Subrouter()
	posts.HandleFunc("/", wrapHandler(true, boardPage))
	posts.HandleFunc(thread, wrapHandler(true, threadPage))

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

	// Return status 500 on goroutine panic
	handler = handlers.RecoveryHandler(handlers.
		PrintRecoveryStack(true),
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

// handles `/board/` page requests
func boardPage(jsonOnly bool, res http.ResponseWriter, req *http.Request) {
	in := indexPage{res: res, req: req, json: jsonOnly}
	board := mux.Vars(req)["board"]

	in.validate = func() bool {
		return canAccessBoard(board, in.ident)
	}

	in.getCounter = func() (counter int) {
		rGet(r.Table("main").
			Get("histCounts").
			Field("board").
			Default(0),
		).
			One(counter)
		return
	}

	in.getPostData = func() []byte {
		data := NewReader(board, in.ident).GetBoard()
		encoded, err := json.Marshal(data)
		throw(err)
		return encoded
	}

	in.process(board)
}

// Handles `/board/thread` requests
func threadPage(jsonOnly bool, res http.ResponseWriter, req *http.Request) {
	in := indexPage{res: res, req: req, json: jsonOnly}
	vars := mux.Vars(req)
	board := vars["board"]
	id, err := strconv.Atoi(vars["thread"])
	throw(err)
	in.lastN = detectLastN(req)

	in.validate = func() bool {
		return validateOP(id, board) && canAccessThread(id, board, in.ident)
	}

	in.getCounter = func() (counter int) {
		rGet(getThread(id).Field("histCtr")).One(&counter)
		return
	}

	in.getPostData = func() []byte {
		data := NewReader(board, in.ident).GetThread(id, in.lastN)
		encoded, err := json.Marshal(data)
		throw(err)
		return encoded
	}

	in.process(board)
}

// Stores common variables anf methods for both board and thread pages
type indexPage struct {
	res         http.ResponseWriter
	req         *http.Request
	validate    func() bool
	getCounter  func() int    // Progress counter used for building etags
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

// Build an etag and check if it mathces the one provided by the client. If yes,
// send 304 and return false, otherwise set headers and return true.
func (in *indexPage) validateEtag() bool {
	etag := in.buildEtag()
	if config.Hard.Debug {
		setHeaders(in.res, noCacheHeaders)
		return true
	}
	hasAuth := in.ident.Auth != ""
	if hasAuth {
		etag += "-" + in.ident.Auth
	}
	if in.lastN != 0 {
		etag += fmt.Sprintf("-last%v", in.lastN)
	}

	// Etags match. No need to rerender.
	if ifNoneMatch, ok := in.req.Header["If-None-Match"]; ok {
		for _, clientEtag := range ifNoneMatch {
			if clientEtag == etag {
				in.res.WriteHeader(304)
				return false
			}
		}
	}

	setHeaders(in.res, vanillaHeaders)
	in.res.Header().Set("ETag", etag)
	if hasAuth {
		in.res.Header().Add("Cache-Control", ", private")
	}
	return true
}

// Build the main part of the etag
func (in *indexPage) buildEtag() string {
	etag := "W/" + strconv.Itoa(in.getCounter())
	if !in.json {
		etag += "-" + in.template.Hash
		if in.isMobile {
			etag += "-mobile"
		}
	}
	return etag
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

func notFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(404)
	custom404(res)
}

func custom404(res http.ResponseWriter) {
	copyFile("www/404.html", res)
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
	data, err := json.Marshal(clientConfig)
	throw(err)
	res.Write(data)
}

// Serve a single post as JSON
func servePost(res http.ResponseWriter, req *http.Request) {
	id, err := strconv.Atoi(mux.Vars(req)["post"])
	throw(err)
	board := parentBoard(id)
	thread := parentThread(id)
	ident := extractIdent(res, req)
	if board == "" || thread == 0 || !canAccessThread(thread, board, ident) {
		res.WriteHeader(404)
		return
	}
	data, err := json.Marshal(NewReader(board, ident).GetPost(id))
	throw(err)
	res.Write(data)
}
