// Synchronized random video playback

package feeds

import (
	"fmt"
	"time"

	"github.com/bakape/meguca/common"

	"github.com/go-playground/log"
)

type video struct {
	FileType uint8         `json:"file_type"`
	Duration time.Duration `json:"-"`
	URL      string        `json:"url"`
}

type tvFeed struct {
	baseFeed
	board     string
	startedAt time.Time
	pos       int
	playList  []video
}

func (f *tvFeed) readPlaylist() (err error) {
	durs := [...]float64{
		1446.001000,
		1446.081000,
		1448.007000,
		1446.001000,
		1448.007000,
		1448.057000,
		1448.007000,
		1445.961000,
		1446.041000,
		1446.001000,
		1446.041000,
		1430.981000,
	}
	f.playList = make([]video, len(durs))
	for i, d := range durs {
		f.playList[i] = video{
			FileType: common.WEBM,
			Duration: time.Duration(float64(time.Second) * d),
			URL:      fmt.Sprintf("/assets/videos/shamiko_%d.webm", i+1),
		}
	}
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
				f.pos++
				f.pos %= 12
				f.startedAt = time.Now()
				f.sendToAll(f.encodePlaylist())
				timer.Reset(f.playList[0].Duration)
			}
		}
	}()

	return
}

func (f *tvFeed) encodePlaylist() []byte {
	current := make([]video, 3)
	for i := 0; i < 3; i++ {
		current[i] = f.playList[(f.pos+i)%12]
	}

	msg, err := common.EncodeMessage(common.MessageMeguTV, struct {
		Elapsed  float64 `json:"elapsed"`
		Playlist []video `json:"playlist"`
	}{
		Elapsed:  time.Now().Sub(f.startedAt).Seconds(),
		Playlist: current,
	})
	if err != nil {
		log.Warnf("video playlist encoding: %s\n", err)
	}
	return msg
}
