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

	// RenderHTML produces HTML from the passed in data and JSON
	RenderHTML func(interface{}, []byte) []byte
}

// GetJSON retrieves JSON from the cache, validates it is still
// fresh or retrieves fresh data, if needed
func GetJSON(k Key, f FrontEnd) ([]byte, uint64, error) {
	s := getStore(k)
	s.Lock()
	defer s.Unlock()

	data, json, ctr, fresh, err := getData(s, f)
	if err != nil {
		return nil, 0, err
	}
	if fresh {
		s.update(data, json, nil)
	}

	return json, ctr, nil
}

func getData(s *store, f FrontEnd) (
	data interface{}, JSON []byte, ctr uint64, fresh bool, err error,
) {
	// Have cached data
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
			s.lastChecked = time.Now().Unix()
			return s.data, s.json, s.updateCounter, false, nil
		}
	}

	fresh = true
	if ctr == 0 {
		ctr, err = f.GetCounter(s.key)
		if err != nil {
			return
		}
		s.updateCounter = ctr
	}
	data, err = f.GetFresh(s.key)
	if err != nil {
		return
	}
	JSON, err = json.Marshal(data)
	if err != nil {
		return
	}
	s.lastChecked = time.Now().Unix()
	return
}

// GetHTML retrieves post HTML from the cache or generates fresh HTML as needed
func GetHTML(k Key, f FrontEnd) ([]byte, uint64, error) {
	s := getStore(k)
	s.Lock()
	defer s.Unlock()

	data, json, ctr, fresh, err := getData(s, f)
	if err != nil {
		return nil, 0, err
	}

	var html []byte
	genHTML := func() {
		html = []byte(f.RenderHTML(data, json))
		s.update(data, json, html)
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

	return html, ctr, nil
}

// ThreadKey encodes a Key from a thread's ID and last N posts to show setting
func ThreadKey(id uint64, lastN int) Key {
	return Key{
		LastN: uint8(lastN),
		ID:    id,
	}
}

// BoardKey encodes a key for a board page resource
func BoardKey(b string) Key {
	return Key{Board: b}
}
