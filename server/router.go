package server

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/util"
	"github.com/bakape/meguca/websockets"
	"github.com/dimfeld/httptreemux"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/go-playground/log"
)

var (
	healthCheckMsg = []byte("God's in His heaven, all's right with the world")
)

// Used for overriding during tests
var webRoot = "www"

func startWebServer() (err error) {
	c := config.Server.Server

	var w strings.Builder
	w.WriteString("listening on http")
	prettyAddr := c.Address
	if len(c.Address) != 0 && c.Address[0] == ':' {
		prettyAddr = "127.0.0.1" + prettyAddr
	}
	fmt.Fprintf(&w, "://%s", prettyAddr)
	log.Info(w.String())

	gracehttp.PreStartProcess(db.Close)
	err = gracehttp.Serve(&http.Server{
		Addr:    c.Address,
		Handler: createRouter(),
	})
	if err != nil {
		return util.WrapError("error starting web server", err)
	}
	return
}

func handlePanic(w http.ResponseWriter, r *http.Request, err interface{}) {
	http.Error(w, fmt.Sprintf("500 %s", err), 500)
	ip, ipErr := auth.GetIP(r)
	if ipErr != nil {
		ip = "invalid IP"
	}
	log.Errorf("server: %s: %#v\n%s\n", ip, err, debug.Stack())
}

// Create the monolithic router for routing HTTP requests. Separated into own
// function for easier testability.
func createRouter() http.Handler {
	r := httptreemux.NewContextMux()
	r.NotFoundHandler = func(w http.ResponseWriter, _ *http.Request) {
		text404(w)
	}
	r.PanicHandler = handlePanic

	r.GET("/robots.txt", serveRobotsTXT)

	api := r.NewGroup("/api")
	api.GET("/health-check", healthCheck)
	assets := r.NewGroup("/assets")
	if config.Server.ImagerMode != config.NoImager {
		// All upload images
		api.POST("/upload", imager.NewImageUpload)
		api.POST("/upload-hash", imager.UploadImageHash)

		assets.GET("/images/*path", serveImages)

		// Captcha API
		captcha := api.NewGroup("/captcha")
		captcha.GET("/:board", serveNewCaptcha)
		captcha.POST("/:board", authenticateCaptcha)
		captcha.GET("/confirmation", renderCaptchaConfirmation)
	}
	if config.Server.ImagerMode != config.ImagerOnly {
		// HTML
		r.GET("/", redirectToDefault)
		r.GET("/:board/", indexHTML)
		r.GET("/:board/catalog", indexHTML)
		r.GET("/:board/:thread", indexHTML)
		r.GET("/all/:id", crossRedirect)

		html := r.NewGroup("/html")
		html.GET("/board-navigation", boardNavigation)
		html.GET("/owned-boards/:userID", ownedBoardSelection)
		html.GET("/create-board", boardCreationForm)
		html.GET("/change-password", changePasswordForm)
		html.POST("/configure-board/:board", boardConfigurationForm)
		html.POST("/configure-server", serverConfigurationForm)
		html.GET("/assign-staff/:board", staffAssignmentForm)
		html.GET("/set-banners", bannerSettingForm)
		html.GET("/set-loading", loadingAnimationForm)
		html.GET("/bans/:board", banList)
		html.GET("/mod-log/:board", modLog)
		html.GET("/report/:id", reportForm)
		html.GET("/reports/:board", reportList)

		// JSON API
		json := r.NewGroup("/json")
		boards := json.NewGroup("/boards")
		boards.GET("/:board/", func(w http.ResponseWriter, r *http.Request) {
			boardJSON(w, r, false)
		})
		boards.GET("/:board/catalog", func(w http.ResponseWriter,
			r *http.Request,
		) {
			boardJSON(w, r, true)
		})
		boards.GET("/:board/:thread", threadJSON)
		json.GET("/post/:post", servePost)
		json.GET("/config", serveConfigs)
		json.GET("/extensions", serveExtensionMap)
		json.GET("/board-config/:board", serveBoardConfigs)
		json.GET("/board-list", serveBoardList)
		json.GET("/ip-count", serveIPCount)
		json.POST("/thread-updates", serveThreadUpdates)
		json.GET("/thread/:thread", threadJSON)

		// Internal API
		api.GET("/socket", func(w http.ResponseWriter, r *http.Request) {
			err := websockets.Handler(w, r)
			if err != nil {
				httpError(w, r, err)
			}
		})
		api.GET("/youtube-data/:id", youTubeData)
		api.GET("/bitchute-title/:id", bitChuteTitle)
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
		api.POST("/delete-posts", deletePosts)
		api.POST("/delete-posts/by-ip", deletePostsByIP)
		api.POST("/delete-image", deleteImage)
		api.POST("/spoiler-image", modSpoilerImage)
		api.POST("/ban", ban)
		api.POST("/notification", sendNotification)
		api.POST("/assign-staff", assignStaff)
		api.POST("/same-IP/:id", getSameIPPosts)
		api.POST("/sticky", setThreadSticky)
		api.POST("/lock-thread", setThreadLock)
		api.POST("/unban/:board", unban)
		api.POST("/set-banners", setBanners)
		api.POST("/set-loading", setLoadingAnimation)
		api.POST("/report", report)
		api.POST("/purge-post", purgePost)
		api.GET("/pprof/:profile", serverProfile)

		redir := api.NewGroup("/redirect")
		redir.POST("/by-ip", redirectByIP)
		redir.POST("/by-thread", redirectByThread)

		// Assets
		assets.GET("/banners/:board/:id", serveBanner)
		assets.GET("/loading/:board", serveLoadingAnimation)
		assets.GET("/*path", serveAssets)
	}

	return r
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

// Redirect the client to the appropriate board through a cross-board redirect
func crossRedirect(w http.ResponseWriter, r *http.Request) {
	idStr := extractParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		text404(w)
		return
	}

	board, op, err := db.GetPostParenthood(id)
	if err != nil {
		httpError(w, r, err)
		return
	}
	url := r.URL
	url.Path = fmt.Sprintf("/%s/%d", board, op)
	if url.Query().Get("last") != "" {
		url.Fragment = "bottom"
	} else {
		url.Fragment = "p" + idStr
	}
	http.Redirect(w, r, url.String(), 301)
}

// Health check to ensure server is still online
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write(healthCheckMsg)
}
