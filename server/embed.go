package server

import (
	"fmt"
	"strings"
	"database/sql"
	"net/http"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"

	"github.com/badoux/goscraper"
	"github.com/xenking/ytdl"
)

// Get YouTube video information by ID
func youTubeData(w http.ResponseWriter, r *http.Request) {
	var info string
	id := extractParam(r, "id")
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		title, thumb, video, videoHigh, err := db.GetYouTubeInfo(tx, id)

		if err != nil {
			title, thumb, video, videoHigh, err = getYouTubeInfo(tx, id)

			if err != nil {
				return err
			}
		}

		info = fmt.Sprintf("%s\n%s\n%s\n%s", title, thumb, video, videoHigh)
		return nil
	})

	if err != nil {
		httpError(w, r, err)
		return
	}

	w.Write([]byte(info))
}

// Get BitChute video title by ID
func bitChuteTitle(w http.ResponseWriter, r *http.Request) {
	var title string
	id := extractParam(r, "id")
	err := db.InTransaction(false, func(tx *sql.Tx) (err error) {
		title, err = db.GetBitChuteTitle(tx, id)

		if err != nil {
			pTitle, err := goscraper.Scrape("https://www.bitchute.com/embed/"+id, 3)

			if err != nil {
				return err
			}

			title = pTitle.Preview.Description
			err = db.WriteBitChuteTitle(tx, id, title)

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		httpError(w, r, err)
		return
	}

	w.Write([]byte(title))
}

// Return YouTube video info and handle errors
func getYouTubeInfo(tx *sql.Tx, id string) (sTitle string, sThumb string, sVideo string, sVideoHigh string, err error) {
	var ok bool
	info, err := ytdl.GetVideoInfoFromID(id)

	if err != nil {
		err = errYouTube(id, err)
		return
	} else if info.Duration == 0 {
		err = errYouTubeLive(id)
		return
	}

	for _, val := range info.Keywords {
		if strings.Contains(val, "live") || strings.Contains(val, "stream") {
			err = errYouTubeLive(id)
			return
		}
	}

	thumb := info.GetThumbnailURL(ytdl.ThumbnailQualityMaxRes)

	for _, val := range [4]ytdl.ThumbnailQuality{
		ytdl.ThumbnailQualityHigh,
		ytdl.ThumbnailQualityMedium,
		ytdl.ThumbnailQualityDefault,
		ytdl.ThumbnailQualitySD,
	} {
		resp, err := http.Get(thumb.String())
		if resp != nil {
			resp.Body.Close()
		}

		if err == nil {
			if resp.StatusCode == http.StatusOK {
				ok = true
				break
			}
		}

		thumb = info.GetThumbnailURL(val)
	}

	if !ok {
		err = errNoYoutubeThumb(id)
		return
	}

	vidFormats := info.Formats.
		Filter(ytdl.FormatExtensionKey, []interface{}{"webm"}).
		Filter(ytdl.FormatResolutionKey, []interface{}{"360p"}).
		Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"}).
		Best(ytdl.FormatVideoEncodingKey)

	if len(vidFormats) == 0 {
		vidFormats = info.Formats.
			Filter(ytdl.FormatExtensionKey, []interface{}{"webm"}).
			Filter(ytdl.FormatResolutionKey, []interface{}{"144p", "240p", "270p", "360p"}).
			Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"}).
			Best(ytdl.FormatResolutionKey).
			Best(ytdl.FormatVideoEncodingKey)

		if len(vidFormats) == 0 {
			vidFormats = info.Formats.
				Filter(ytdl.FormatExtensionKey, []interface{}{"mp4", "flv", "3gp", "ts"}).
				Filter(ytdl.FormatResolutionKey, []interface{}{"144p", "240p", "270p", "360p"}).
				Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"}).
				Best(ytdl.FormatResolutionKey).
				Best(ytdl.FormatVideoEncodingKey)

			if len(vidFormats) == 0 {
				err = errNoYoutubeVideo(id)
				return
			}
		}
	}

	video, err := info.GetDownloadURL(vidFormats[0])

	if err != nil {
		err = errYouTube(id, err)
		return
	}

	// Unfortunately, in some cases you cannot get 720p with only webm
	vidFormats = info.Formats.
		Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"}).
		Best(ytdl.FormatResolutionKey).
		Best(ytdl.FormatVideoEncodingKey)

	if len(vidFormats) == 0 {
		err = errNoYoutubeVideo(id)
		return
	}

	videoHigh, err := info.GetDownloadURL(vidFormats[0])

	if err != nil {
		err = errYouTube(id, err)
		return
	}

	sTitle = info.Title
	sThumb = strings.Replace(thumb.String(), "http://", "https://", 1)
	sVideo = video.String()
	sVideoHigh = videoHigh.String()

	err = db.WriteYouTubeInfo(tx, id, sTitle, sThumb, sVideo, sVideoHigh)

	if err != nil {
		err = errYouTube(id, err)
		return
	}

	return
}

func errYouTube(id string, err error) error {
	return errYouTubeGeneric(id, err.Error(), 500)
}

func errYouTubeLive(id string) error {
	return errYouTubeGeneric(id, "Video is a livestream", 415)
}

func errNoYoutubeVideo(id string) error {
	return errYouTubeGeneric(id, "Video does not exist", 404)
}

func errNoYoutubeThumb(id string) error {
	return errYouTubeGeneric(id, "Thumbnail does not exist", 404)
}

func errYouTubeGeneric(id string, err string, code int) error {
	return common.StatusError{fmt.Errorf("YouTube [%s]: %s", id, err), code}
}
