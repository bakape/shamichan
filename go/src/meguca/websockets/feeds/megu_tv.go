// Synchronized random video playback

package feeds

import (
	"database/sql"
	"meguca/common"
	"meguca/db"
	"time"
)

type tvFeed struct {
	baseFeed
	sha1, board string
	startedAt   time.Time
	duration    time.Duration
}

func (f *tvFeed) readVideo() (err error) {
	var length uint
	f.sha1, length, err = db.RandomVideo(f.board)
	if err != nil {
		return
	}
	f.duration = time.Second * time.Duration(length)
	f.startedAt = time.Now()
	return
}

func (f *tvFeed) start(board string) (err error) {
	f.board = board
	err = f.readVideo()
	switch err {
	case nil:
	case sql.ErrNoRows:
		err = nil
	default:
		return err
	}

	go func() {
		timer := time.NewTimer(f.duration)
		defer timer.Stop()

		for {
			select {
			case c := <-f.add:
				f.addClient(c)
				msg := f.encodeMessage()
				if msg != nil {
					c.Send(msg)
				}
			case c := <-f.remove:
				if f.removeClient(c) {
					return
				}
			case <-timer.C:
				switch err := f.readVideo(); err {
				case nil:
					msg := f.encodeMessage()
					if msg != nil {
						for _, c := range f.clients {
							c.Send(msg)
						}
					}
					timer.Reset(f.duration)
				case sql.ErrNoRows:
				default:

				}
			}
		}
	}()

	return
}

func (f *tvFeed) encodeMessage() []byte {
	if f.sha1 == "" {
		return nil
	}
	msg, _ := common.EncodeMessage(common.MessageMeguTV, struct {
		Elapsed uint   `json:"elapsed"`
		Sha1    string `json:"sha1"`
	}{
		Elapsed: uint(time.Now().Sub(f.startedAt).Seconds()),
		Sha1:    f.sha1,
	})
	return msg
}
