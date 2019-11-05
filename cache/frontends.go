package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/db"
	"github.com/Chiiruno/meguca/templates"
)

// ErrPageOverflow is an error detailing that the page was not found
var ErrPageOverflow = errors.New("page not found")

// PageStore contains data of a board page
type PageStore struct {
	PageNumber int
	JSON       []byte
	Data       common.Board
}

// ThreadFE is for accessing cached thread pages
var ThreadFE = FrontEnd{
	GetCounter: func(k Key) (uint64, error) {
		return db.ThreadCounter(k.ID)
	},

	GetFresh: func(k Key) (interface{}, error) {
		return db.GetThread(k.ID, int(k.LastN))
	},

	RenderHTML: func(data interface{}, json []byte) []byte {
		var b bytes.Buffer
		templates.WriteThreadPosts(&b, data.(common.Thread), json)
		return b.Bytes()
	},
}

// CatalogFE is for accessing cached catalog pages
var CatalogFE = FrontEnd{
	GetCounter: func(k Key) (uint64, error) {
		if k.Board == "all" {
			return db.AllBoardCounter()
		}
		return db.BoardCounter(k.Board)
	},

	GetFresh: func(k Key) (interface{}, error) {
		if k.Board == "all" {
			return db.GetAllBoardCatalog()
		}
		return db.GetBoardCatalog(k.Board)
	},

	RenderHTML: func(data interface{}, json []byte) []byte {
		var b bytes.Buffer
		templates.WriteCatalogThreads(&b, data.(common.Board).Threads, json)
		return b.Bytes()
	},
}

// BoardFE is for accessing cached board pages
var BoardFE = FrontEnd{
	GetCounter: func(k Key) (uint64, error) {
		if k.Board == "all" {
			return db.AllBoardCounter()
		}
		return db.BoardCounter(k.Board)
	},

	// Board pages are built as a list of individually fetched and cached
	// threads with up to 5 replies each
	GetFresh: func(k Key) (interface{}, error) {
		// Get thread IDs in the right order
		var (
			ids []uint64
			err error
		)
		if k.Board == "all" {
			ids, err = db.GetAllThreadsIDs()
		} else {
			ids, err = db.GetThreadIDs(k.Board)
		}
		if err != nil {
			return nil, err
		}

		// Empty board
		if len(ids) == 0 {
			data := common.Board{Threads: []common.Thread{}}
			buf, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			return []PageStore{
				{
					PageNumber: 1,
					JSON:       buf,
					Data:       data,
				},
			}, nil
		}

		// Get data and JSON for these views and paginate
		var (
			pages = make([]PageStore, 0, len(ids)/15+1)
			page  PageStore
		)
		closePage := func() {
			if page.Data.Threads != nil {
				pages = append(pages, page)
			}
		}

		// Hide threads from NSFW boards, if enabled
		var (
			confs    map[string]config.BoardConfContainer
			hideNSFW bool
		)
		if k.Board == "all" && config.Get().HideNSFW {
			hideNSFW = true
			confs = config.GetAllBoardConfigs()
		}

		for i, id := range ids {
			// Start a new page
			if i%15 == 0 {
				closePage()
				page = PageStore{
					PageNumber: len(pages),
					Data: common.Board{
						Threads: make([]common.Thread, 0, 15),
					},
				}
			}

			k := ThreadKey(id, 5)
			_, data, _, err := GetJSONAndData(k, ThreadFE)
			if err != nil {
				return nil, err
			}
			t := data.(common.Thread)

			if hideNSFW && confs[t.Board].NSFW {
				continue
			}

			page.Data.Threads = append(page.Data.Threads, t)
		}
		closePage()

		// Record total page count in all stores and generate JSON
		l := len(pages)
		if l == 0 { // Empty board
			l = 1
			pages = []PageStore{
				{
					JSON: []byte(`{"threads":[],"pages":1}`),
				},
			}
		}
		for i := range pages {
			p := &pages[i]
			p.Data.Pages = l
			p.JSON, err = json.Marshal(p.Data)
			if err != nil {
				return nil, err
			}
		}

		return pages, nil
	},

	Size: func(data interface{}, _, _ []byte) (s int) {
		for _, p := range data.([]PageStore) {
			s += len(p.JSON) * 2
		}
		return
	},
}

// BoardPageFE is for individual pages of a board index page
var BoardPageFE = FrontEnd{
	GetCounter: func(k Key) (uint64, error) {
		// Get the counter of the parent board
		k.Page = -1
		_, _, ctr, err := GetJSONAndData(k, BoardFE)
		return ctr, err
	},

	GetFresh: func(k Key) (interface{}, error) {
		i := int(k.Page)
		k.Page = -1
		_, data, _, err := GetJSONAndData(k, BoardFE)
		if err != nil {
			return nil, err
		}

		pages := data.([]PageStore)
		if i > len(pages)-1 {
			return nil, ErrPageOverflow
		}
		return pages[i], nil
	},

	EncodeJSON: func(data interface{}) ([]byte, error) {
		return data.(PageStore).JSON, nil
	},

	RenderHTML: func(data interface{}, json []byte) []byte {
		var b bytes.Buffer
		templates.WriteIndexThreads(&b, data.(PageStore).Data.Threads, json)
		return b.Bytes()
	},

	Size: func(_ interface{}, _, html []byte) int {
		// Only the HTML is owned by this store. All other data is just
		// borrowed from board
		return len(html)
	},
}
