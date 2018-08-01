package server

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/imager"
	"meguca/util"
	"meguca/websockets"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/dimfeld/httptreemux"
	"github.com/go-playground/log"
	"github.com/gorilla/handlers"
	"github.com/otium/ytdl"
)

var (
	// Address is the listening address of the HTTP web server
	address string

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
	log.Info("listening on " + address)

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
	if config.ImagerMode != config.NoImager {
		// All upload images
		api.POST("/upload", imager.NewImageUpload)
		api.POST("/upload-hash", imager.UploadImageHash)
		api.POST("/create-thread", createThread)
		api.POST("/create-reply", createReply)

		assets.GET("/images/*path", serveImages)
	}
	if config.ImagerMode != config.ImagerOnly {
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
		json.GET("/watch", serveThreadWatcher)

		// Internal API
		api.GET("/socket", websockets.Handler)
		api.GET("/youtube-data/:id", youTubeData)
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
		api.POST("/lock-thread", setThreadLock)
		api.POST("/unban/:board", unban)
		api.POST("/set-banners", setBanners)
		api.POST("/set-loading", setLoadingAnimation)
		api.POST("/report", report)

		redir := api.NewGroup("/redirect")
		redir.POST("/by-ip", redirectByIP)
		redir.POST("/by-thread", redirectByThread)

		// Captcha API
		captcha := api.NewGroup("/captcha")
		captcha.GET("/new", db.NewCaptchaID)
		captcha.GET("/image/*path", db.ServeCaptcha)

		// Noscript captcha API
		NSCaptcha := captcha.NewGroup("/noscript")
		NSCaptcha.GET("/load", noscriptCaptchaLink)
		NSCaptcha.GET("/new", noscriptCaptcha)

		// Assets
		assets.GET("/banners/:board/:id", serveBanner)
		assets.GET("/loading/:board", serveLoadingAnimation)
		assets.GET("/*path", serveAssets)
	}

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
	w.Write([]byte("God's in His heaven, all's right with the world"))
}

// Get YouTube title and googlevideo URL from URL
func youTubeData(w http.ResponseWriter, r *http.Request) {
	ytid := extractParam(r, "id")
	code, err := func() (code uint16, err error) {
		code = 500
		info, err := ytdl.GetVideoInfoFromID(ytid)

		if err != nil {
			return
		} else if info.Duration == 0 {
			return errYouTubeLive(ytid)
		}

		vidFormats := info.Formats.
			Filter(ytdl.FormatExtensionKey, []interface{}{"webm"}).
			Filter(ytdl.FormatResolutionKey, []interface{}{"360p"}).
			Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"})

		if len(vidFormats) == 0 {
			vidFormats = info.Formats.
				Filter(ytdl.FormatExtensionKey, []interface{}{"webm"}).
				Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"}).
				Worst(ytdl.FormatResolutionKey)

			if len(vidFormats) == 0 {
				return errNoYoutubeVideo(ytid)
			}
		}

		video, err := info.GetDownloadURL(vidFormats[0])

		if err != nil {
			return
		}

		// Unfortunately, in some cases you cannot get 720p with only webm
		vidFormats = info.Formats.
			Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"}).
			Best(ytdl.FormatResolutionKey)

		if len(vidFormats) == 0 {
			return errNoYoutubeVideo(ytid)
		}

		videoHigh, err := info.GetDownloadURL(vidFormats[0])

		if err != nil {
			return
		}

		thumb := info.GetThumbnailURL(ytdl.ThumbnailQualityMaxRes)

		for i := 0; i < 5; i++ {
			ok, err := func() (bool, error) {
				// Perhaps there is a way to check the status code without fetching the body?
				resp, err := http.Get(thumb.String())

				if err != nil {
					return false, err
				}

				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					return true, err
				}

				return false, err
			}()

			if err != nil {
				return errNoYoutubeThumb(ytid)
			}

			if !ok {
				switch i {
				case 0:
					thumb = info.GetThumbnailURL(ytdl.ThumbnailQualityHigh)
				case 1:
					thumb = info.GetThumbnailURL(ytdl.ThumbnailQualityMedium)
				case 2:
					thumb = info.GetThumbnailURL(ytdl.ThumbnailQualityDefault)
				case 3:
					thumb = info.GetThumbnailURL(ytdl.ThumbnailQualitySD)
				default:
					return errNoYoutubeThumb(ytid)
				}
			} else {
				break
			}
		}

		fmt.Fprintf(w, "%s\n%s\n%s\n%s",
			info.Title,
			strings.Replace(thumb.String(), "http://", "https://", 1),
			video.String(),
			videoHigh.String(),
		)

		return 200, nil
	}()

	if err != nil {
		if !common.CanIgnoreClientError(err) {
			err = common.StatusError{
				fmt.Errorf("YouTube fetch error on ID `%s` %s", ytid, err),
				int(code),
			}
		}

		httpError(w, r, err)
	}
}

func errYouTubeLive(id string) (uint16, error) {
	return 415, common.StatusError{errors.New("YouTube video [" + id + "] is a livestream"), 415}
}

func errNoYoutubeVideo(id string) (uint16, error) {
	return 404, common.StatusError{errors.New("YouTube video [" + id + "] does not exist"), 404}
}

func errNoYoutubeThumb(id string) (uint16, error) {
	return 404, common.StatusError{errors.New("YouTube thumbnail [" + id + "] does not exist"), 404}
}
