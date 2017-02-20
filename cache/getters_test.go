package cache

import (
	"testing"
	"time"

	. "../test"
)

func init() {
	expiryTime = 1
}

func TestGetJSON(t *testing.T) {
	Clear()

	var fetches, counterChecks int
	key := ThreadKey(33, 3)
	f := FrontEnd{
		GetCounter: func(k Key) (uint64, error) {
			counterChecks++
			if k == key {
				return 1, nil
			}
			return 0, nil
		},
		GetFresh: func(k Key) (interface{}, error) {
			fetches++
			if k == key {
				return []string{"foo"}, nil
			}
			return nil, nil
		},
	}

	for i := 0; i < 2; i++ {
		json, ctr, err := GetJSON(key, f)
		if err := err; err != nil {
			t.Fatal(err)
		}
		AssertDeepEquals(t, string(json), `["foo"]`)
		AssertDeepEquals(t, ctr, uint64(1))
	}
	assertCount(t, "fetched", 1, fetches)
	assertCount(t, "counter checked", 1, counterChecks)
}

func assertCount(t *testing.T, action string, std, n int) {
	if n != std {
		t.Errorf("%s too many times: %d", action, n)
	}
}

func TestGetHTML(t *testing.T) {
	Clear()

	var fetches, renders int
	f := FrontEnd{
		GetCounter: func(k Key) (uint64, error) {
			return 1, nil
		},
		GetFresh: func(k Key) (interface{}, error) {
			fetches++
			return "foo", nil
		},
		RenderHTML: func(_ interface{}, _ []byte) []byte {
			renders++
			return []byte("bar")
		},
	}

	for i := 0; i < 2; i++ {
		json, ctr, err := GetHTML(BoardKey("a", false), f)
		if err := err; err != nil {
			t.Fatal(err)
		}
		AssertDeepEquals(t, string(json), `bar`)
		AssertDeepEquals(t, ctr, uint64(1))
	}
	assertCount(t, "fetched", 1, fetches)
	assertCount(t, "rendered", 1, fetches)

	t.Run("with json", func(t *testing.T) {
		key := BoardKey("c", false)

		if _, _, err := GetJSON(key, f); err != nil {
			t.Fatal(err)
		}
		if _, _, err := GetHTML(key, f); err != nil {
			t.Fatal(err)
		}

		assertCount(t, "fetched", 2, fetches)
		assertCount(t, "rendered", 2, fetches)
	})
}

func TestCounterExpiry(t *testing.T) {
	Clear()

	var counterChecks, fetches int
	f := FrontEnd{
		GetCounter: func(k Key) (uint64, error) {
			counterChecks++
			return 1, nil
		},
		GetFresh: func(k Key) (interface{}, error) {
			fetches++
			return "foo", nil
		},
	}

	k := BoardKey("a", false)
	if _, _, err := GetJSON(k, f); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Duration(float64(time.Second) * 1.1))
	if _, _, err := GetJSON(k, f); err != nil {
		t.Fatal(err)
	}

	assertCount(t, "fetches", 1, fetches)
	assertCount(t, "counter checks", 2, counterChecks)
}
