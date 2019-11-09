package auth

import (
	"time"

	"github.com/bakape/meguca/common"
	"github.com/go-playground/log"
)

// ModLogEntry is a single entry in the moderation log
type ModLogEntry struct {
	common.ModerationEntry
	ID      uint64    `json:"id"`
	Created time.Time `json:"created"`
	Board   string    `json:"board"`
}

// Ban holds an entry of an IP being banned from a board
type Ban struct {
	IP, Board string
}

// BanRecord stores information about a specific ban
type BanRecord struct {
	Ban
	ForPost          uint64
	Reason, By, Type string
	Expires          time.Time
}

// Report contains data of a reported post
type Report struct {
	ID, Target    uint64
	Created       time.Time
	Board, Reason string
}

// DisconnectByBoardAndIP disconnects all banned
// websocket clients matching IP from board.
// /all/ board disconnects all clients globally.
func DisconnectByBoardAndIP(ip, board string) {
	msg, err := common.EncodeMessage(common.MessageInvalid,
		common.ErrBanned.Error())
	if err != nil {
		log.Error(err)
		return
	}
	var cls []common.Client
	if board == "all" {
		cls = common.GetClientsByIP(ip)
	} else {
		cls = common.GetByIPAndBoard(ip, board)
	}
	for _, cl := range cls {
		cl.Send(msg)
		cl.Close(nil)
	}
}
