package common

var (
	modLevelStr = [...]string{
		"",
		"janitors",
		"moderators",
		"owners",
		"admin",
	}
	modActionStr = [...]string{
		"ban_post",
		"unban_post",
		"delete_post",
		"delete_image",
		"spoiler_image",
		"lock_thread",
		"delete_board",
		"meido_vision",
		"purge_post",
		"shadow_bin_post",
	}
)

// ModerationAction is an action performable by moderation staff
type ModerationAction uint8

func (m ModerationAction) MarshalText() (text []byte, err error) {
	return []byte(modActionStr[m]), nil
}

func (m *ModerationAction) UnmarshalText(text []byte) error {
	s := string(text)
	for i, a := range modActionStr {
		if s == a {
			*m = ModerationAction(i)
			return nil
		}
	}
	return ErrInvalidEnum(s)
}

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
	PurgePost
	ShadowBinPost
)

// Contains fields of a post moderation log entry
type ModerationEntry struct {
	Type   ModerationAction `json:"type"`
	Length uint64           `json:"length"`
	By     string           `json:"by"`
	Data   string           `json:"data"`
}

// ModerationLevel defines the level required to perform an action or the
// permission level held by a user
type ModerationLevel int8

// Returns string representation of moderation level
func (l ModerationLevel) String() string {
	if l < Janitor {
		return ""
	}
	return modLevelStr[int(l)]
}

func (m ModerationLevel) MarshalText() (text []byte, err error) {
	return []byte(modLevelStr[m]), nil
}

func (m *ModerationLevel) UnmarshalText(text []byte) error {
	s := string(text)
	for i, a := range modActionStr {
		if s == a {
			*m = ModerationLevel(i)
			return nil
		}
	}
	return ErrInvalidEnum(s)
}

// All available moderation levels
const (
	NotStaff ModerationLevel = iota - 1
	Janitor
	Moderator
	BoardOwner
	Admin
)
