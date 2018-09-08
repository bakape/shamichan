package common

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

// Contains fields of a post moderation log entry
type ModerationEntry struct {
	Type   ModerationAction `json:"type"`
	Length uint64           `json:"length"`
	By     string           `json:"by"`
	Reason string           `json:"reason"`
}
