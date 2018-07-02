package feeds

import (
	"encoding/json"
	"meguca/common"
	"time"
)

// Persists thread state for syncing clients to server feed
type threadCache struct {
	posts map[uint64]common.Post
}

type threadMeta struct {
	id        uint64
	Sticky    bool   `json:"sticky"`
	NonLive   bool   `json:"nonLive"`
	Locked    bool   `json:"locked"`
	PostCtr   uint32 `json:"postCtr"`
	ImageCtr  uint32 `json:"imageCtr"`
	ReplyTime int64  `json:"replyTime"`
	BumpTime  int64  `json:"bumpTime"`
	Subject   string `json:"subject"`
	Board     string `json:"board"`
}

// Extract cache data from common.Thread.
// TODO: Remove this mapping, once C++ client is in production
func newThreadCache(t common.Thread) threadCache {
	cap := len(t.Posts) * 2
	c := threadCache{
		posts: make(map[uint64]common.Post, cap),
	}
	c.posts[t.ID] = t.Post
	for _, p := range t.Posts {
		c.posts[p.ID] = p
	}

	return c
}

// Message used for synchronizing clients to the feed state.
// This is the version used by the current JS client.
type syncMessage struct {
	Recent       []uint64            `json:"recent"`
	Banned       []uint64            `json:"banned"`
	Deleted      []uint64            `json:"deleted"`
	DeletedImage []uint64            `json:"deletedImage"`
	Open         map[uint64]openPost `json:"open"`
}

type openPost struct {
	HasImage  bool   `json:"hasImage"`
	Spoilered bool   `json:"spoilered"`
	Body      string `json:"body"`
}

// Generate a message for synchronizing to the current status of the update
// feed. The client has to compare this state to it's own and resolve any
// missing entries or conflicts.
func (c *threadCache) genSyncMessage() []byte {
	threshold := time.Now().Add(-time.Minute * 15).Unix()
	msg := syncMessage{
		Recent:       make([]uint64, 0, 16),
		Banned:       make([]uint64, 0, 16),
		Deleted:      make([]uint64, 0, 16),
		DeletedImage: make([]uint64, 0, 16),
		Open:         make(map[uint64]openPost, 16),
	}
	for id, p := range c.posts {
		if p.Time > threshold {
			msg.Recent = append(msg.Recent, id)
		}
		if p.Editing {
			op := openPost{
				HasImage: p.Image != nil,
				Body:     p.Body,
			}
			if op.HasImage {
				op.Spoilered = p.Image.Spoiler
			}
			msg.Open[id] = op
		}
		if p.Deleted {
			msg.Deleted = append(msg.Deleted, id)
		}
		if p.Banned {
			msg.Banned = append(msg.Banned, id)
		}
	}

	buf, _ := json.Marshal(msg)
	return common.PrependMessageType(common.MessageSynchronise, buf)
}
