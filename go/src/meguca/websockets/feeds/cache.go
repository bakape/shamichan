package feeds

import (
	"encoding/json"
	"meguca/common"
	"sort"
	"strconv"
	"time"

	"github.com/bakape/thumbnailer"
	"github.com/mailru/easyjson"
)

// Persists thread state for syncing clients to server feed
type threadCache struct {
	threadMeta
	Posts    map[uint64]common.Post
	memoized map[uint64][]byte
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
		threadMeta: threadMeta{
			id:        t.ID,
			Sticky:    t.Sticky,
			NonLive:   t.NonLive,
			Locked:    t.Locked,
			PostCtr:   t.PostCtr,
			ImageCtr:  t.ImageCtr,
			ReplyTime: t.ReplyTime,
			BumpTime:  t.BumpTime,
			Subject:   t.Subject,
			Board:     t.Board,
		},
		Posts:    make(map[uint64]common.Post, cap),
		memoized: make(map[uint64][]byte, cap),
	}
	c.Posts[t.ID] = t.Post
	for _, p := range t.Posts {
		c.Posts[p.ID] = p
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

// As syncMessage, but used for the newer protocol with C++ clients
type cppSyncMessage struct {
	threadMeta
	Posts []*common.Post `json:"posts"`
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
	for id, p := range c.Posts {
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

type uintSorter []uint64

func (u uintSorter) Len() int {
	return len(u)
}

func (u uintSorter) Less(i, j int) bool {
	return u[i] < u[j]
}

func (u uintSorter) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

func (c *threadCache) encodeThread(last100 bool) []byte {
	// Map is randomly ordered, so need to map IDs and sort
	ids := make([]uint64, 0, len(c.Posts))
	for id, _ := range c.Posts {
		ids = append(ids, id)
	}
	sort.Sort(uintSorter(ids))

	if last100 {
		i := len(ids) - 100
		if i > 0 {
			// Keep OP in the array
			sliced := make([]uint64, 100)
			sliced[0] = ids[0]
			copy(sliced[1:], ids[i:])
			ids = sliced
		}
	}

	b := thumbnailer.GetBuffer()
	b = append(b, `30{"sticky":`...)
	b = strconv.AppendBool(b, c.Sticky)
	b = append(b, `,"nonLive":`...)
	b = strconv.AppendBool(b, c.NonLive)
	b = append(b, `,"locked":`...)
	b = strconv.AppendBool(b, c.Locked)
	b = append(b, `,"deleted":`...)
	b = strconv.AppendBool(b, c.Posts[c.id].Deleted)
	b = append(b, `,"postCtr":`...)
	b = strconv.AppendUint(b, uint64(c.PostCtr), 10)
	b = append(b, `,"imageCtr":`...)
	b = strconv.AppendUint(b, uint64(c.ImageCtr), 10)
	b = append(b, `,"time":`...)
	b = strconv.AppendInt(b, c.Posts[c.id].Time, 10)
	b = append(b, `,"replyTime":`...)
	b = strconv.AppendInt(b, c.ReplyTime, 10)
	b = append(b, `,"bumpTime":`...)
	b = strconv.AppendInt(b, c.BumpTime, 10)
	b = append(b, `,"subject":`...)
	b = strconv.AppendQuote(b, c.Subject)
	b = append(b, `,"board":`...)
	b = strconv.AppendQuote(b, c.Board)
	b = append(b, `,"posts":[`...)
	for i, id := range ids {
		if i != 0 {
			b = append(b, ',')
		}

		mem, ok := c.memoized[id]
		if !ok {
			mem, _ = easyjson.Marshal(c.Posts[id])
			c.memoized[id] = mem
		}
		b = append(b, mem...)
	}
	b = append(b, "]}"...)
	return b
}

// Clear memoized post JSON, if any
func (c *threadCache) deleteMemoized(id uint64) {
	delete(c.memoized, id)
}
