// Synchronized random video playback

package feeds

import (
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"time"

	"github.com/go-playground/log"
)

type tvFeed struct {
	baseFeed
	board     string
	startedAt time.Time
	playList  []db.Video
}

func (f *tvFeed) readPlaylist() (err error) {
	f.playList, err = db.VideoPlaylist(f.board)
	return
}

func (f *tvFeed) start(board string) (err error) {
	f.board = board
	err = f.readPlaylist()
	if err != nil {
		return
	}

	go func() {
		dur := time.Hour // In case there are no videos
		if len(f.playList) != 0 {
			dur = f.playList[0].Duration
		}
		f.startedAt = time.Now()
		timer := time.NewTimer(dur)
		defer timer.Stop()

		for {
			select {
			case c := <-f.add:
				f.addClient(c)
				c.Send(f.encodePlaylist())
			case c := <-f.remove:
				if f.removeClient(c) {
					return
				}
			case <-timer.C:
				// Refetch playlist, if too short or file missing
				needFetch := false
				if len(f.playList) < 2 {
					needFetch = true
				} else {
					var visible bool
					visible, err = db.ImageVisible(f.playList[1].SHA1, board)
					if err != nil {
						log.Warnf("verifying video is visible: %s\n", err)
						continue
					}
					needFetch = !visible
				}
				if needFetch {
					err := f.readPlaylist()
					if err != nil {
						log.Warnf("fetching video playlist: %s\n", err)
						continue
					}
				} else {
					// Otherwise decrease list by one
					f.playList = f.playList[1:]
				}
				f.startedAt = time.Now()
				f.sendToAll(f.encodePlaylist())
				dur := time.Hour // If empty playlist
				if len(f.playList) != 0 {
					dur = f.playList[0].Duration
				}
				timer.Reset(dur)
			}
		}
	}()

	return
}

func (f *tvFeed) encodePlaylist() []byte {
	i := 2
	if len(f.playList) < 2 {
		i = len(f.playList)
	}
	msg, err := common.EncodeMessage(common.MessageMeguTV, struct {
		Elapsed  float64    `json:"elapsed"`
		Playlist []db.Video `json:"playlist"`
	}{
		Elapsed:  time.Now().Sub(f.startedAt).Seconds(),
		Playlist: f.playList[:i],
	})
	if err != nil {
		log.Warnf("video playlist encoding: %s\n", err)
	}
	return msg
}
