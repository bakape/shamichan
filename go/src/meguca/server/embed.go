package server

import (
	"fmt"
	"errors"
	"strings"
	"net/http"

	"meguca/common"

	"github.com/otium/ytdl"
	"github.com/badoux/goscraper"
)

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

// Get BitChute title from ID
func bitChuteTitle(w http.ResponseWriter, r *http.Request) {
	s, err := goscraper.Scrape("https://www.bitchute.com/embed/" + extractParam(r, "id"), 3)

	if err != nil {
		httpError(w, r, err)
		return
	}

	w.Write([]byte(s.Preview.Description))
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
