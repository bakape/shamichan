package auth

import (
	"meguca/common"
	"time"

	"github.com/go-playground/log"
)

// ModerationLevel defines the level required to perform an action
type ModerationLevel int8

// FromString reads moderation level from string representation
func (l *ModerationLevel) FromString(s string) {
	switch s {
	case "admin":
		*l = Admin
	case "owners":
		*l = BoardOwner
	case "moderators":
		*l = Moderator
	case "janitors":
		*l = Janitor
	default:
		*l = NotStaff
	}
}

// Returns string representation of moderation level
func (l ModerationLevel) String() string {
	switch l {
	case Admin:
		return "admin"
	case BoardOwner:
		return "owners"
	case Moderator:
		return "moderators"
	case Janitor:
		return "janitors"
	default:
		return ""
	}
}

// All available moderation levels
const (
	NotLoggedIn ModerationLevel = iota - 1
	NotStaff
	Janitor
	Moderator
	BoardOwner
	Admin
)

// ModerationAction is an action performable by moderation staff
type ModerationAction uint8

// All supported moderation actions
const (
	BanPost ModerationAction = iota
	UnbanPost
	DeletePost
	DeleteImage
	SpoilerImage
	LockThread
	DeleteBoard
	MeidoVision
)

// ModLogEntry is a single entry in the moderation log
type ModLogEntry struct {
	Type              ModerationAction `json:"type"`
	ID                uint64           `json:"id"`
	Length            uint64           `json:"length"`
	Created           time.Time        `json:"created"`
	Board             string           `json:"board"`
	By                string           `json:"by"`
	Reason            string           `json:"reason"`
}

// Ban holdsan entry of an IP being banned from a board
type Ban struct {
	IP, Board string
}

// BanRecord stores information about a specific ban
type BanRecord struct {
	Ban
	ForPost    uint64
	Reason, By string
	Expires    time.Time
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
