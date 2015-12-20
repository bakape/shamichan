/*
 Webserver
*/

package main

import (
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func startServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", redirectToDefault)
	router.HandleFunc(`/{board:\w+}`, addTrailingSlash)
	//sub := router.Path(`/{board:\w+}/`).Subrouter()

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

func boardPage(res http.ResponseWriter, req http.Request) {

}
