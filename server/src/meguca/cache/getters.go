package cache

import (
	"encoding/json"
	"time"
)

// FrontEnd provides functions for fetching, validating and generating the
// cache. GetCounter and GetFresh are mandatory, but you may omit RenderHTML, if
// you don't plan to call GetHTML with this FrontEnd.
type FrontEnd struct {
	// GetCounter retrieves a new update counter for a resource
	GetCounter func(Key) (uint64, error)

	// GetFresh retrieves new post data from the database
	GetFresh func(Key) (interface{}, error)

	// Encode data into JSON. If null, default encoder is used.
	EncodeJSON func(data interface{}) ([]byte, error)

	// RenderHTML produces HTML from the passed in data and JSON
	RenderHTML func(interface{}, []byte) []byte

	// Calculates the size taken by the store.
	// If nil, the default function is used.
	Size func(data interface{}, json, html []byte) int
}

// GetJSONAndData GetJSON retrieves JSON from the cache along with unencoded post data,
// validates, if is still fresh, or retrieves fresh data, if needed
func GetJSONAndData(k Key, f FrontEnd) ([]byte, interface{}, uint64, error) {
	s := getStore(k)
	s.Lock()
	defer s.Unlock()

	data, json, ctr, fresh, err := getData(s, f)
	if err != nil {
		return nil, nil, 0, err
	}
	if fresh {
		s.update(data, json, nil, f)
	}

	return json, data, ctr, nil
}

func getData(s *store, f FrontEnd) (
	data interface{}, buf []byte, ctr uint64, fresh bool, err error,
) {
	// Have cached data and json
	if s.data != nil {
		if s.isFresh() {
			// No freshness check needed yet
			return s.data, s.json, s.updateCounter, false, nil
		}
		ctr, err = f.GetCounter(s.key)
		if err != nil {
			return
		}
		if ctr == s.updateCounter {
			// Still fresh
			s.lastChecked = time.Now()
			return s.data, s.json, s.updateCounter, false, nil
		}
	}

	fresh = true
	if ctr == 0 {
		ctr, err = f.GetCounter(s.key)
		if err != nil {
			return
		}
	}
	s.updateCounter = ctr
	data, err = f.GetFresh(s.key)
	if err != nil {
		return
	}

	if f.EncodeJSON != nil {
		buf, err = f.EncodeJSON(data)
	} else {
		buf, err = json.Marshal(data)
	}
	if err != nil {
		return
	}

	s.lastChecked = time.Now()
	return
}

// GetHTML retrieves post HTML from the cache or generates fresh HTML as needed
func GetHTML(k Key, f FrontEnd) ([]byte, interface{}, uint64, error) {
	s := getStore(k)
	s.Lock()
	defer s.Unlock()

	data, json, ctr, fresh, err := getData(s, f)
	if err != nil {
		return nil, nil, 0, err
	}

	var html []byte
	genHTML := func() {
		html = []byte(f.RenderHTML(data, json))
		s.update(data, json, html, f)
	}
	if !fresh {
		// If the cache has been filled with a JSON request, it will not have
		// any HTML
		if s.html != nil {
			html = s.html
		} else {
			genHTML()
		}
	} else {
		genHTML()
	}

	return html, data, ctr, nil
}

// ThreadKey encodes a Key from a thread's ID and last N posts to show setting
func ThreadKey(id uint64, lastN int) Key {
	return Key{
		LastN: uint8(lastN),
		ID:    id,
	}
}

// BoardKey encodes a key for a board page resource
func BoardKey(b string, page int64, index bool) Key {
	// Index theads will have a lastN == 1
	lastN := uint8(0)
	if index {
		lastN = 1
	}
	return Key{
		Board: b,
		Page:  page,
		LastN: lastN,
	}
}
