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

	// Defines if HTTPS should be used for listening for incoming connections.
	// Requires sslCert and sslKey to be set.
	ssl bool

	// Path to SSL certificate
	sslCert string

	// Path to SSL key
	sslKey string

	// Defines, if all traffic should be piped through a gzip compression
	// -decompression handler
	enableGzip bool

	isTest bool
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
	r.GET("/:board/", boardHTML)
	r.GET("/:board/:thread", threadHTML)

	// API for retrieving various localized HTML forms
	forms := r.NewGroup("/forms")
	forms.GET("/boardNavigation", wrapHandler(boardNavigation))
	forms.GET("/ownedBoards/:userID", ownedBoardSelection)
	forms.GET("/createBoard", wrapHandler(boardCreationForm))
	forms.GET("/changePassword", wrapHandler(changePasswordForm))
	forms.GET("/captcha", wrapHandler(renderCaptcha))
	forms.POST("/configureBoard", wrapHandler(boardConfigurationForm))
	forms.POST("/configureServer", wrapHandler(serverConfigurationForm))

	// JSON API
	json := r.NewGroup("/json")
	json.GET("/:board/", boardJSON)
	json.GET("/:board/:thread", threadJSON)
	json.GET("/post/:post", servePost)
	json.GET("/config", wrapHandler(serveConfigs))
	json.GET("/extensions", wrapHandler(serveExtensionMap))
	json.GET("/boardConfig/:board", serveBoardConfigs)
	json.GET("/boardList", wrapHandler(serveBoardList))
	json.GET("/positions/:position/:user", serveStaffPositions)

	// Public POST API
	r.POST("/createThread", wrapHandler(createThread))
	r.POST("/spoiler", wrapHandler(spoilerImage))

	// Administration JSON API for logged in users
	admin := r.NewGroup("/admin")
	admin.POST("/register", wrapHandler(register))
	admin.POST("/login", wrapHandler(login))
	admin.POST("/logout", wrapHandler(logout))
	admin.POST("/logoutAll", wrapHandler(logoutAll))
	admin.POST("/changePassword", wrapHandler(changePassword))
	admin.POST("/boardConfig", wrapHandler(servePrivateBoardConfigs))
	admin.POST("/configureBoard", wrapHandler(configureBoard))
	admin.POST("/config", wrapHandler(servePrivateServerConfigs))
	admin.POST("/configureServer", wrapHandler(configureServer))
	admin.POST("/createBoard", wrapHandler(createBoard))
	admin.POST("/deleteBoard", wrapHandler(deleteBoard))

	// Assets
	r.GET("/assets/*path", serveAssets)
	r.GET("/images/*path", serveImages)
	r.GET("/worker.js", wrapHandler(serveWorker))

	// Websocket API
	r.GET("/socket", wrapHandler(websockets.Handler))

	// File upload
	r.POST("/upload", wrapHandler(imager.NewImageUpload))
	r.POST("/uploadHash", wrapHandler(imager.UploadImageHash))

	h := http.Handler(r)
	if enableGzip {
		h = handlers.CompressHandlerLevel(h, gzip.DefaultCompression)
	}

	return h
}

// Adapter for http.HandlerFunc -> httptreemux.HandlerFunc
func wrapHandler(fn http.HandlerFunc) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		fn(w, r)
	}
}

// Redirects to / requests to /all/ board
func redirectToDefault(res http.ResponseWriter, req *http.Request) {
	http.Redirect(res, req, "/all/", 302)
}
