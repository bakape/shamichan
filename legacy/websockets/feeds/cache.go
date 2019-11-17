package feeds

import (
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
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
	c.All, c.Open, c.Moderation, err = db.GetThreadMeta(id)
	return
}

// Message used for synchronizing clients to the feed state.
type syncMessage struct {
	// All posts in thread as (post_id, page) map
	All        map[uint64]uint32                   `json:"all"`
	Open       map[uint64]db.OpenPostMeta          `json:"open"`
	Moderation map[uint64][]common.ModerationEntry `json:"moderation"`
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
	c.memoized, err = common.EncodeMessage(
		common.MessageSynchronise,
		c.syncMessage,
	)
	return c.memoized, err
}

// Clear memoized sync message JSON, if any
func (c *threadCache) clearMemoized() {
	c.memoized = nil
}
