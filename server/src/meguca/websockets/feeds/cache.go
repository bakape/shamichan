package feeds

import (
	"meguca/common"
)

// Persists thread state for syncing clients to server feed
type threadCache struct {
	syncMessage
	memoized []byte
}

func newThreadCache(id uint64) (c threadCache, err error) {
	c = threadCache{
		syncMessage: syncMessage{
			Recent:     make(map[uint64]cachedPost),
			Moderation: make(map[uint64][]common.ModerationEntry),
		},
	}
	// TODO: Read data from DB
	return
}

// Message used for synchronizing clients to the feed state.
type syncMessage struct {
	Recent     map[uint64]cachedPost               `json:"recent"`
	Moderation map[uint64][]common.ModerationEntry `json:"moderation"`
}

type cachedPost struct {
	HasImage  bool   `json:"has_image"`
	Spoilered bool   `json:"spoilered"`
	Closed    bool   `json:"closed"`
	Time      int64  `json:"time"`
	Body      string `json:"body"`
}

// Generate a message for synchronizing to the current status of the update
// feed. The client has to compare this state to it's own and resolve any
// missing entries or conflicts.
//
// Returned buffer must not be modified.
func (c *threadCache) getSyncMessage() ([]byte, error) {
	if c.memoized != nil {
		return c.memoized, nil
	}

	var err error
	c.memoized, err = common.EncodeMessage(common.MessageSynchronise,
		c.syncMessage)
	// Strip newline
	if len(c.memoized) != 0 {
		c.memoized = c.memoized[:len(c.memoized)-1]
	}
	return c.memoized, err
}

// Clear memoized sync message JSON, if any
func (c *threadCache) clearMemoized() {
	c.memoized = nil
}
