package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"meguca/auth"
	"meguca/config"
	"meguca/imager"
	"meguca/util"
	"meguca/websockets"
	"net/http"

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
	r := httptreemux.NewContextMux()
	r.NotFoundHandler = func(w http.ResponseWriter, _ *http.Request) {
		text404(w)
	}
	r.PanicHandler = text500

	r.GET("/robots.txt", serveRobotsTXT)

	// HTML
	r.GET("/", redirectToDefault)
	r.GET("/:board/", func(w http.ResponseWriter, r *http.Request) {
		boardHTML(w, r, false)
	})
	r.GET("/:board/catalog", func(w http.ResponseWriter, r *http.Request) {
		boardHTML(w, r, true)
	})
	// Needs override, because it conflicts with crossRedirect
	r.GET("/all/catalog", func(w http.ResponseWriter, r *http.Request) {
		// Artificially set board to "all"
		r = r.WithContext(context.WithValue(
			r.Context(),
			httptreemux.ParamsContextKey,
			map[string]string{
				"board": "all",
			},
		))
		boardHTML(w, r, true)
	})
	r.GET("/:board/:thread", threadHTML)
	r.GET("/all/:id", crossRedirect)

	// API for retrieving various localized HTML forms
	forms := r.NewGroup("/forms")
	forms.GET("/boardNavigation", boardNavigation)
	forms.GET("/ownedBoards/:userID", ownedBoardSelection)
	forms.GET("/createBoard", boardCreationForm)
	forms.GET("/changePassword", changePasswordForm)
	forms.GET("/captcha", renderCaptcha)
	forms.POST("/configureBoard/:board", boardConfigurationForm)
	forms.POST("/configureServer", serverConfigurationForm)
	forms.GET("/assignStaff/:board", staffAssignmentForm)
	forms.GET("/setBanners", bannerSettingForm)

	// JSON API
	json := r.NewGroup("/json")
	json.GET("/:board/", func(w http.ResponseWriter, r *http.Request) {
		boardJSON(w, r, false)
	})
	json.GET("/:board/catalog", func(w http.ResponseWriter, r *http.Request) {
		boardJSON(w, r, true)
	})
	json.GET("/:board/:thread", threadJSON)
	json.GET("/post/:post", servePost)
	json.GET("/config", serveConfigs)
	json.GET("/extensions", serveExtensionMap)
	json.GET("/boardConfig/:board", serveBoardConfigs)
	json.GET("/boardList", serveBoardList)

	// Public POST API
	r.POST("/createThread", createThread)
	r.POST("/createReply", createReply)

	// Administration JSON API for logged in users
	admin := r.NewGroup("/admin")
	admin.POST("/register", register)
	admin.POST("/login", login)
	admin.POST("/logout", logout)
	admin.POST("/logoutAll", logoutAll)
	admin.POST("/changePassword", changePassword)
	admin.POST("/boardConfig/:board", servePrivateBoardConfigs)
	admin.POST("/configureBoard/:board", configureBoard)
	admin.POST("/config", servePrivateServerConfigs)
	admin.POST("/configureServer", configureServer)
	admin.POST("/createBoard", createBoard)
	admin.POST("/deleteBoard", deleteBoard)
	admin.POST("/deletePost", deletePost)
	admin.POST("/deleteImage", deleteImage)
	admin.POST("/spoilerImage", modSpoilerImage)
	admin.POST("/ban", ban)
	admin.POST("/notification", sendNotification)
	admin.POST("/assignStaff", assignStaff)
	admin.POST("/sameIP/:id", getSameIPPosts)
	admin.POST("/sticky", setThreadSticky)
	admin.POST("/unban/:board", unban)
	admin.POST("/setBanners", setBanners)

	// Available to both logged-in users and publicly with slight alterations
	r.GET("/bans/:board", banList)
	r.GET("/mod-log/:board", modLog)

	// Captcha API
	captcha := r.NewGroup("/captcha")
	captcha.GET("/new", auth.NewCaptchaID)
	captcha.GET("/image/*path", auth.ServeCaptcha)

	// Noscript captcha API
	NSCaptcha := captcha.NewGroup("/noscript")
	NSCaptcha.GET("/load", noscriptCaptchaLink)
	NSCaptcha.GET("/new", noscriptCaptcha)

	// Assets
	r.GET("/assets/*path", serveAssets)
	r.GET("/images/*path", serveImages)
	r.GET("/worker.js", serveWorker)

	// Websocket API
	r.GET("/socket", websockets.Handler)

	// File upload
	r.POST("/upload", imager.NewImageUpload)
	r.POST("/uploadHash", imager.UploadImageHash)

	h := http.Handler(r)
	if enableGzip {
		h = handlers.CompressHandlerLevel(h, gzip.DefaultCompression)
	}

	return h
}

// Redirects to / requests to /all/ board
func redirectToDefault(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/all/", 301)
}

// Generate a robots.txt with only select boards preventing indexing
func serveRobotsTXT(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	// Would be pointles without the /all/ board disallowed.
	// Also, this board can be huge. Don't want bots needlessly crawling it.
	buf.WriteString("User-agent: *\nDisallow: /all/\n")
	for _, c := range config.GetAllBoardConfigs() {
		if c.DisableRobots {
			fmt.Fprintf(&buf, "Disallow: /%s/\n", c.ID)
		}
	}
	w.Header().Set("Content-Type", "text/plain")
	buf.WriteTo(w)
}
