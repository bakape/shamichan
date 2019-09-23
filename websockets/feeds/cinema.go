package feeds

import (
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/util"
	"time"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"strconv"

	"github.com/go-playground/log"
)

type cinemaStatus struct {
	watching int
	playing string
}

type cinemaFeed struct {
	baseFeed
	mainFeed *Feed
	thread uint64
	playlist []cinemaVideo
	startTime time.Time
	push chan string
	voteSkip chan string
	skippers map[string]struct{}
	videoTimer *time.Timer
	status cinemaStatus
}

type cinemaVideo struct {
	Url string `json:"url"`
	Duration int64 `json:"duration"`
	Title string `json:"title"`
	Type string `json:"type"`
}

type cinemaMessage struct {
	Cmd  string      `json:"cmd"`
	Data interface{} `json:"data"`
}

func PushToCinema(thread uint64, url string, ip string) {
	feeds.mu.RLock()
	defer feeds.mu.RUnlock()
	if feed := feeds.cinemaFeeds[thread]; feed != nil {
		feed.push <- url
	}
}

func CinemaVoteSkip(thread uint64, ip string) {
	feeds.mu.RLock()
	defer feeds.mu.RUnlock()
	if feed := feeds.cinemaFeeds[thread]; feed != nil {
		feed.voteSkip <- ip
	}
}

func newCinemaFeed(thread uint64, mainFeed *Feed) *cinemaFeed {
	var cf = &cinemaFeed{}
	cf.thread = thread
	cf.mainFeed = mainFeed
	cf.push = make(chan string)
	cf.voteSkip = make(chan string)
	cf.skippers = make(map[string]struct{})
	// expired timer to avoid null dereference in select
	cf.videoTimer = time.NewTimer(time.Second*1)
	cf.videoTimer.Stop()
	cf.init()
	return cf
}

func (f *cinemaFeed) start() (err error) {
	go func() {
		ticker := time.NewTicker(time.Second*5)
		defer ticker.Stop()

		for {
			select {
			case c := <-f.add:
				f.addClient(c)
				c.Send(f.envelopPlaylist())
				f.updateStatus()
			case c := <-f.remove:
				cIP := c.IP()
				if f.removeClient(c) { // end yourself if last client left
					f.playlist = f.playlist[:0]
					f.updateStatus()
					return
				}
				// update skippers list
				if _, ok := f.skippers[cIP] ; ok {
					ipActive := false
					for otherC := range f.clients {
						if cIP == otherC.IP() {
							ipActive = true
							break
						}
					}
					if !ipActive {
						delete(f.skippers, cIP)
					}
				}
				// send status info
				f.updateStatus()
			case <-ticker.C:
				if len(f.playlist) > 0 {
					f.sendToAll(f.envelopSyncTime())
				}
			case url := <-f.push:
				if len(f.playlist) > 128 {
					continue
				}
				cv, err := parseUrl(url)
				if err == nil {
					f.playlist = append(f.playlist, cv)
					if len(f.playlist) == 1 { // start seance
						f.videoTimer.Reset(time.Duration(cv.Duration)*time.Millisecond)
						f.startTime = time.Now()
					}
					f.sendToAll(f.envelopPush())
					f.updateStatus()
				} else {
					log.Warnf("Error parsing cinema url: %s\n", err)
				}
			case <-f.videoTimer.C: // end of video
				f.pop()
			case voterIP := <-f.voteSkip:
				if len(f.playlist) == 0 {
					continue
				}
				isSpectator := false
				for c := range f.clients {
					if voterIP == c.IP() {
						isSpectator = true
						break
					}
				}
				if isSpectator {
					f.skippers[voterIP] = struct{}{}
					if len(f.skippers)*2 > f.uniqueIPs() {
						f.pop()
					}
				}
			}
		}
	}()
	return
}

// updates status from cinema feed thread
func (f *cinemaFeed) updateStatus() {
	f.status.watching = f.uniqueIPs()
	if len(f.playlist) > 0 {
		f.status.playing = f.playlist[0].Title
	} else {
		f.status.playing = ""
	}
	// send trigger to feed thread to send status update
	select {
	case f.mainFeed.sendIPCountChan <- f.status:
	default:
		log.Warnf("could not send cinema status update, channel buffer overflow\n")
	}
}

func (f *cinemaFeed) pop() {
	f.skippers = make(map[string]struct{})
	f.playlist = f.playlist[1:]
	f.sendToAll(f.envelopPop())
	f.videoTimer.Stop()
	// start next
	if len(f.playlist) > 0 {
		cv := f.playlist[0]
		f.videoTimer.Reset(time.Duration(cv.Duration)*time.Millisecond)
		f.startTime = time.Now()
	}
	f.updateStatus()
}

func envelopMessage(m cinemaMessage) []byte {
	cinemaMessageJson, err := common.EncodeMessage(common.MessageCinemaSubscription, m)
	if err != nil {
		log.Warnf("video playlist encoding: %s\n", err)
	}
	return cinemaMessageJson
}

func (f *cinemaFeed) envelopPush() []byte {
	m := cinemaMessage{"push", struct {
		Video cinemaVideo `json:"video"`
	}{f.playlist[len(f.playlist)-1]}}
	return envelopMessage(m)
}

func (f *cinemaFeed) envelopPlaylist() []byte {
	m := cinemaMessage{"playlist", struct {
		Playlist []cinemaVideo `json:"playlist"`
		CurrentTime int64 `json:"currentTime"`
	}{
		f.playlist,
		time.Now().Sub(f.startTime).Milliseconds(),
	}}
	return envelopMessage(m)
}

func (f *cinemaFeed) envelopSyncTime() []byte {
	m := cinemaMessage{"syncTime", struct {
		CurrentTime int64 `json:"currentTime"`
	}{time.Now().Sub(f.startTime).Milliseconds()}}
	return envelopMessage(m)
}

func (f *cinemaFeed) envelopPop() []byte {
	m := cinemaMessage{"pop", nil}
	return envelopMessage(m)
}

type invidiousRes struct {
	LengthSeconds int64 `json:"lengthSeconds"`
	Title string `json:"title"`
}

func getInfoInvidious(url string) (cv cinemaVideo, err error) {
	var res *http.Response
	id := common.InvidiousUrlRegexp.FindStringSubmatch(url)[1]
	res, err = http.Get("https://invidio.us/api/v1/videos/"+id+"?fields=lengthSeconds,title")
	if err != nil {
		return
	}
	ir := &invidiousRes{}
	err = json.NewDecoder(res.Body).Decode(ir)
	if err != nil {
		return
	}
	cv.Url = url
	cv.Title = ir.Title
	cv.Duration = ir.LengthSeconds*1000
	cv.Type = "invidious"
	return
}

func ffprobeGetMediaDuration(file string) (durationMilliseconds int64, err error) {
	args := []string{
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		file,
	}
	var out []byte
	out, err = util.ExecBinary("ffprobe", args, 8000*time.Millisecond)
	if err != nil {
		return
	}
	var durationSeconds float64
	durationSeconds, err = strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return
	}
	return int64(durationSeconds*1000), nil
}

func getInfoRawFile(url string) (cv cinemaVideo, err error) {
	var duration int64
		duration, err = ffprobeGetMediaDuration(url)
	if err != nil {
		return
	}
	cv.Url = url
	cv.Title = url
	cv.Duration = duration
	cv.Type = "raw"
	return
}

func getInfoYoutube(url string) (cv cinemaVideo, err error) {
	args := []string{
		"-j", url,
	}
	var out []byte
	out, err = util.ExecBinary("youtube-dl", args, 8000*time.Millisecond)
	if err != nil {
		return
	}
	// ytdl output also has duration and title fields
	json.Unmarshal(out, &cv)

	cv.Url = url
	cv.Duration *= 1000
	cv.Type = "youtube"
	return
}

func parseUrl(url string) (cv cinemaVideo, err error) {
	switch {
	case common.InvidiousUrlRegexp.MatchString(url):
		cv, err = getInfoInvidious(url)
	case common.GetRawVideoUrlRegexp().MatchString(url):
		cv, err = getInfoRawFile(url)
	case common.YoutubeUrlRegexp.MatchString(url):
		cv, err = getInfoYoutube(url)
	default:
		err = errors.New("no matching url pattern")
		return
	}
	if err == nil && cv.Duration == 0 {
		err = errors.New("couldn't obtain video duration")
	}
	return
}
