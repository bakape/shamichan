package server

import (
	"bytes"
	"compress/gzip"
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
		boardHTML(w, r, extractParam(r, "board"), false)
	})
	r.GET("/:board/catalog", func(w http.ResponseWriter, r *http.Request) {
		boardHTML(w, r, extractParam(r, "board"), true)
	})
	// Needs override, because it conflicts with crossRedirect
	r.GET("/all/catalog", func(w http.ResponseWriter, r *http.Request) {
		// Artificially set board to "all"
		boardHTML(w, r, "all", true)
	})
	r.GET("/:board/:thread", threadHTML)
	r.GET("/all/:id", crossRedirect)

	html := r.NewGroup("/html")
	html.GET("/board-navigation", boardNavigation)
	html.GET("/owned-boards/:userID", ownedBoardSelection)
	html.GET("/create-board", boardCreationForm)
	html.GET("/change-password", changePasswordForm)
	html.GET("/captcha", renderCaptcha)
	html.POST("/configure-board/:board", boardConfigurationForm)
	html.POST("/configure-server", serverConfigurationForm)
	html.GET("/assign-staff/:board", staffAssignmentForm)
	html.GET("/set-banners", bannerSettingForm)
	html.GET("/bans/:board", banList)
	html.GET("/mod-log/:board", modLog)

	// JSON API
	json := r.NewGroup("/json")
	boards := json.NewGroup("/boards")
	boards.GET("/:board/", func(w http.ResponseWriter, r *http.Request) {
		boardJSON(w, r, false)
	})
	boards.GET("/:board/catalog", func(w http.ResponseWriter, r *http.Request) {
		boardJSON(w, r, true)
	})
	boards.GET("/:board/:thread", threadJSON)
	json.GET("/post/:post", servePost)
	json.GET("/config", serveConfigs)
	json.GET("/extensions", serveExtensionMap)
	json.GET("/board-config/:board", serveBoardConfigs)
	json.GET("/board-list", serveBoardList)

	// Internal API
	api := r.NewGroup("/api")
	api.GET("/socket", websockets.Handler)
	api.POST("/upload", imager.NewImageUpload)
	api.POST("/upload-hash", imager.UploadImageHash)
	api.POST("/create-thread", createThread)
	api.POST("/create-reply", createReply)
	api.POST("/register", register)
	api.POST("/login", login)
	api.POST("/logout", logout)
	api.POST("/logout-all", logoutAll)
	api.POST("/change-password", changePassword)
	api.POST("/board-config/:board", servePrivateBoardConfigs)
	api.POST("/configure-board/:board", configureBoard)
	api.POST("/config", servePrivateServerConfigs)
	api.POST("/configure-server", configureServer)
	api.POST("/create-board", createBoard)
	api.POST("/delete-board", deleteBoard)
	api.POST("/delete-post", deletePost)
	api.POST("/delete-image", deleteImage)
	api.POST("/spoiler-image", modSpoilerImage)
	api.POST("/ban", ban)
	api.POST("/notification", sendNotification)
	api.POST("/assign-staff", assignStaff)
	api.POST("/same-IP/:id", getSameIPPosts)
	api.POST("/sticky", setThreadSticky)
	api.POST("/unban/:board", unban)
	api.POST("/set-banners", setBanners)

	// Captcha API
	captcha := api.NewGroup("/captcha")
	captcha.GET("/new", auth.NewCaptchaID)
	captcha.GET("/image/*path", auth.ServeCaptcha)

	// Noscript captcha API
	NSCaptcha := captcha.NewGroup("/noscript")
	NSCaptcha.GET("/load", noscriptCaptchaLink)
	NSCaptcha.GET("/new", noscriptCaptcha)

	// Assets
	assets := r.NewGroup("/assets")
	assets.GET("/banners/:board/:id", serveBanner)
	assets.GET("/*path", serveAssets)
	assets.GET("/images/*path", serveImages)
	r.GET("/worker.js", serveWorker)

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
