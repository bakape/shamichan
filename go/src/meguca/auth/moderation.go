package auth

import (
	"meguca/common"
	"sync"
	"time"
)

var (
	// board: IP: IsBanned
	bans   = map[string]map[string]bool{}
	bansMu sync.RWMutex
)

// ModerationLevel defines the level required to perform an action
type ModerationLevel int8

// Reads moderation level from string representation
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

// An action performable by moderation staff
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
)

// Single entry in the moderation log
type ModLogEntry struct {
	Type              ModerationAction
	ID, Length        uint64
	Created           time.Time
	Board, By, Reason string
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

// Contains data of a reported post
type Report struct {
	ID, Target    uint64
	Created       time.Time
	Board, Reason string
}

// IsBanned returns if the IP is banned on the target board
func IsBanned(board, ip string) (banned bool) {
	bansMu.RLock()
	defer bansMu.RUnlock()
	global := bans["all"]
	ips := bans[board]

	if global != nil && global[ip] {
		return true
	}
	if ips != nil && ips[ip] {
		return true
	}
	return false
}

// GetBannedLevels is like IsBanned, but returns, if the IP is banned globally
// or only from the specific board.
func GetBannedLevels(board, ip string) (globally, locally bool) {
	bansMu.RLock()
	defer bansMu.RUnlock()
	global := bans["all"]
	ips := bans[board]
	return global != nil && global[ip], ips != nil && ips[ip]
}

// SetBans replaces the ban cache with the new set
func SetBans(b ...Ban) {
	new := map[string]map[string]bool{}
	for _, b := range b {
		board, ok := new[b.Board]
		if !ok {
			board = map[string]bool{}
			new[b.Board] = board
		}
		board[b.IP] = true
	}

	bansMu.Lock()
	bans = new
	bansMu.Unlock()
}

// Disconnect all banned websocket clients matching IP from board.
// /all/ board disconnects all clients globally.
func DisconnectBannedIP(ip, board string) {
	msg, err := common.EncodeMessage(common.MessageInvalid,
		common.ErrBanned.Error())
	if err != nil {
		panic(err)
	}
	for _, cl := range common.GetByIPAndBoard(ip, board) {
		cl.Send(msg)
		cl.Close(nil)
	}
}
