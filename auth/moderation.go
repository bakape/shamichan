package auth

import (
	"net"
	"time"

	"github.com/bakape/meguca/common"
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
	IP     net.IP `json:"ban"`
	Thread uint64 `json:"thread"` // 0, if global ban
}

// BanRecord stores information about a specific ban
type BanRecord struct {
	Ban
	ForPost uint64    `json:"for_post"`
	Reason  string    `json:"reason"`
	By      string    `json:"by"`
	Type    string    `json:"type"`
	Expires time.Time `json:"expires"`
}

// Report contains data of a reported post
type Report struct {
	ID      uint64    `json:"id"`
	Target  uint64    `json:"target"`
	Created time.Time `json:"created"`
	Board   string    `json:"board"`
	Reason  string    `json:"reason"`
}
