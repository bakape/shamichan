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
	"github.com/sebest/xff"
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

	// Defines, if the server should interpret X-Forwarded-For headers as the
	// actual IP of the request
	trustProxies bool

	// Defines, if all trafic should be piped through a gzip compression
	// -decompression handler
	enableGzip bool
)

// Used for overriding during tests
var (
	webRoot      = "www"
	imageWebRoot = "img"
)

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
	r.NotFoundHandler = text404
	r.PanicHandler = textErrorPage

	// HTML
	r.GET("/", wrapHandler(redirectToDefault))
	r.GET("/all/", wrapHandler(serveIndexTemplate))
	r.GET("/:board/", boardHTML)
	r.GET("/:board/:thread", threadHTML)

	// JSON API
	r.GET("/json/all/", wrapHandler(allBoardJSON))
	r.GET("/json/:board/", boardJSON)
	r.GET("/json/:board/:thread", threadJSON)
	r.GET("/json/config", wrapHandler(serveConfigs))
	r.GET("/json/post/:post", servePost)
	r.GET("/json/boardConfig/:board", serveBoardConfigs)

	// Assets
	assetServer = http.FileServer(http.Dir(webRoot))
	r.GET("/assets/*path", serveAssets)
	r.GET("/images/*path", serveImages)
	r.GET("/worker.js", wrapHandler(serveWorker))

	// Websocket API
	r.GET("/socket", wrapHandler(websockets.Handler))

	// Image upload
	r.POST("/upload", wrapHandler(imager.NewImageUpload))

	h := http.Handler(r)
	if enableGzip {
		h = handlers.CompressHandlerLevel(h, gzip.DefaultCompression)
	}
	if trustProxies {
		xffParser, err := xff.Default()
		if err != nil {
			log.Fatal(err)
		}
		h = xffParser.Handler(h)
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
