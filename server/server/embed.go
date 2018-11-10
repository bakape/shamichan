package server

import (
	"fmt"
	"strings"
	"net/http"

	"meguca/common"

	"github.com/otium/ytdl"
	"github.com/badoux/goscraper"
)

// Get YouTube video information by ID
func youTubeData(w http.ResponseWriter, r *http.Request) {
	info, err := getYouTubeInfo(extractParam(r, "id"))

	if err != nil {
		httpError(w, r, err)
		return
	}

	w.Write([]byte(info))
}

// Get BitChute video title by ID
func bitChuteTitle(w http.ResponseWriter, r *http.Request) {
	s, err := goscraper.Scrape("https://www.bitchute.com/embed/" + extractParam(r, "id"), 3)

	if err != nil {
		httpError(w, r, err)
		return
	}

	w.Write([]byte(s.Preview.Description))
}

// Return YouTube video info and handle errors
func getYouTubeInfo(id string) (ret string, err error) {
	var ok bool
	info, err := ytdl.GetVideoInfoFromID(id)

	if err != nil {
		return ret, errYouTube(id, err)
	} else if info.Duration == 0 {
		return ret, errYouTubeLive(id)
	}

	for _, val := range info.Keywords {
		if strings.Contains(val, "live") || strings.Contains(val, "stream") {
			return ret, errYouTubeLive(id)
		}
	}

	thumb := info.GetThumbnailURL(ytdl.ThumbnailQualityMaxRes)

	for _, val := range [4]ytdl.ThumbnailQuality {
		ytdl.ThumbnailQualityHigh,
		ytdl.ThumbnailQualityMedium,
		ytdl.ThumbnailQualityDefault,
		ytdl.ThumbnailQualitySD,
	} {
		resp, err := http.Get(thumb.String())
		resp.Body.Close()

		if err == nil {
			if resp.StatusCode == http.StatusOK {
				ok = true
				break
			}
		}

		thumb = info.GetThumbnailURL(val)
	}

	if !ok {
		return ret, errNoYoutubeThumb(id)
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
				return ret, errNoYoutubeVideo(id)
			}
		}
	}

	video, err := info.GetDownloadURL(vidFormats[0])

	if err != nil {
		return ret, errYouTube(id, err)
	}

	// Unfortunately, in some cases you cannot get 720p with only webm
	vidFormats = info.Formats.
		Filter(ytdl.FormatAudioEncodingKey, []interface{}{"aac", "opus", "vorbis"}).
		Best(ytdl.FormatResolutionKey).
		Best(ytdl.FormatVideoEncodingKey)

	if len(vidFormats) == 0 {
		return ret, errNoYoutubeVideo(id)
	}

	videoHigh, err := info.GetDownloadURL(vidFormats[0])

	if err != nil {
		return ret, errYouTube(id, err)
	}

	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		info.Title,
		strings.Replace(thumb.String(), "http://", "https://", 1),
		video.String(),
		videoHigh.String(),
	), nil
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
