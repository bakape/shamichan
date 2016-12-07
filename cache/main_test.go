package cache

import "testing"

// Basic test for deadlocks
func TestConcurrency(t *testing.T) {
	Clear()

	f := FrontEnd{
		GetCounter: func(k Key) (uint64, error) {
			return 1, nil
		},
		GetFresh: func(k Key) (interface{}, error) {
			return "foo", nil
		},
	}

	for i := 0; i < 3; i++ {
		go func() {
			for j := 0; j < 100; i++ {
				go func(j int) {
					k := ThreadKey(uint64(j), 0)
					if _, _, err := GetJSON(k, f); err != nil {
						t.Fatal(err)
					}
				}(j)
			}
		}()
	}
}

func TestCacheEviction(t *testing.T) {
	Clear()

	CacheSize = 10
	f := FrontEnd{
		GetCounter: func(k Key) (uint64, error) {
			return 1, nil
		},
		GetFresh: func(k Key) (interface{}, error) {
			return "ab", nil
		},
	}

	for i := 0; i < 3; i++ {
		if _, _, err := GetJSON(ThreadKey(uint64(i), 0), f); err != nil {
			t.Fatal(err)
		}
	}

	_, ok := cache[ThreadKey(0, 0)]
	if ok {
		t.Error("store not evicted")
	}
}
