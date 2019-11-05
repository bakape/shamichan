package feeds

import (
	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/db"
	"time"
)

// One minute higher than post open limit, to reduce border cases
const retentionTime = 16 * time.Minute

// Persists thread state for syncing clients to server feed
type threadCache struct {
	syncMessage
	memoized []byte
}

func retentionThreshold() int64 {
	return time.Now().Add(-retentionTime).Unix()
}

func newThreadCache(id uint64) (c threadCache, err error) {
	c = threadCache{
		syncMessage: syncMessage{
			Recent:     make(map[uint64]cachedPost, 16),
			Moderation: make(map[uint64][]common.ModerationEntry, 16),
		},
	}
	thread, err := db.GetThread(id, 0)
	if err != nil {
		return
	}

	threshold := retentionThreshold()
	for _, p := range thread.Posts {
		if p.Time > threshold {
			c.Recent[p.ID] = cachedPost{
				HasImage:  p.Image != nil,
				Spoilered: p.Image != nil && p.Image.Spoiler,
				Closed:    !p.Editing,
				Time:      p.Time,
				Body:      p.Body,
			}
		}
		if p.Moderated {
			c.Moderation[p.ID] = p.Moderation
		}
	}

	// TODO: Clean up the cache periodically
	return
}

// Evict posts past evictionLimit
func (c *threadCache) evict() {
	c.clearMemoized()
	threshold := retentionThreshold()
	for id, p := range c.Recent {
		if p.Time < threshold {
			delete(c.Recent, id)
		}
	}
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
	Time      int64  `json:"-"`
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
	return c.memoized, err
}

// Clear memoized sync message JSON, if any
func (c *threadCache) clearMemoized() {
	c.memoized = nil
}
