package server

import (
	"compress/gzip"
	"log"
	"net/http"

	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/server/websockets"
	"github.com/bakape/meguca/util"
	"github.com/dimfeld/httptreemux"
	"github.com/gorilla/handlers"
)

var (
	// Address is the listening address of the HTTP web server
	address = ":8000"

	// Defines if HTTPS should be used for listening for incomming connections.
	// Requires sslCert and sslKey to be set.
	ssl bool

	// Path to SSL certificate
	sslCert string

	// Path to SSL key
	sslKey string

	// Defines, if all trafic should be piped through a gzip compression
	// -decompression handler
	enableGzip bool
)

// Used for overriding during tests
var webRoot = "www"

func startWebServer() (err error) {
	r := createRouter()
	log.Println("listening on " + address)

	if ssl {
		err = http.ListenAndServeTLS(address, sslCert, sslKey, r)
	} else {
		err = http.ListenAndServe(address, r)
	}
	if err != nil {
		return util.WrapError("error starting web server", err)
	}
	return
}

// Create the monolithic router for routing HTTP requests. Separated into own
// function for easier testability.
func createRouter() http.Handler {
	r := httptreemux.New()
	r.NotFoundHandler = func(w http.ResponseWriter, _ *http.Request) {
		text404(w)
	}
	r.PanicHandler = text500

	// HTML
	r.GET("/", wrapHandler(redirectToDefault))
	r.GET("/all/", wrapHandler(serveIndexTemplate))
	r.GET("/:board/", boardHTML)
	r.GET("/:board/:thread", threadHTML)

	// JSON API
	json := r.NewGroup("/json")
	json.GET("/all/", wrapHandler(allBoardJSON))
	json.GET("/:board/", boardJSON)
	json.GET("/:board/:thread", threadJSON)
	json.GET("/post/:post", servePost)
	json.GET("/config", wrapHandler(serveConfigs))
	json.GET("/boardConfig/:board", serveBoardConfigs)
	json.GET("/boardList", wrapHandler(serveBoardList))
	json.GET("/positions/:position/:user", serveStaffPositions)

	// Adminitration JSON API for logged in users
	admin := r.NewGroup("/admin")
	admin.POST("/configureBoard", wrapHandler(configureBoard))
	admin.POST("/boardConfig", wrapHandler(servePrivateBoardConfigs))

	// Assets
	assetServer = http.FileServer(http.Dir(webRoot))
	r.GET("/assets/*path", serveAssets)
	r.GET("/images/*path", serveImages)
	r.GET("/worker.js", wrapHandler(serveWorker))

	// Websocket API
	r.GET("/socket", wrapHandler(websockets.Handler))

	// File upload
	r.POST("/upload", wrapHandler(imager.NewImageUpload))

	h := http.Handler(r)
	if enableGzip {
		h = handlers.CompressHandlerLevel(h, gzip.DefaultCompression)
	}

	return h
}

// Adapter for http.HandlerFunc -> httptreemux.HandlerFunc
func wrapHandler(fn http.HandlerFunc) httptreemux.HandlerFunc {
	return func(
		res http.ResponseWriter,
		req *http.Request,
		_ map[string]string,
	) {
		fn(res, req)
	}
}

// Redirects to / requests to /all/ board
func redirectToDefault(res http.ResponseWriter, req *http.Request) {
	http.Redirect(res, req, "/all/", 302)
}
