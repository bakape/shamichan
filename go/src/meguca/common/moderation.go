package common

import "time"

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
	Type              ModerationAction
	ID, Length        uint64
	Created           time.Time
	Board, By, Reason string
}
