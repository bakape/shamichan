package cache

import (
	"sync"
	"testing"

	. "github.com/bakape/meguca/test"
)

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

	var wg sync.WaitGroup
	wg.Add(300)
	for i := 0; i < 3; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				go func(j int) {
					defer wg.Done()
					k := ThreadKey(uint64(j), 0)
					if _, _, err := GetJSON(k, f); err != nil {
						t.Fatal(err)
					}
				}(j)
			}
		}()
	}
	wg.Wait()
}

func TestCacheEviction(t *testing.T) {
	Clear()

	Size = 0.005
	f := FrontEnd{
		GetCounter: func(k Key) (uint64, error) {
			return 1, nil
		},
		GetFresh: func(k Key) (interface{}, error) {
			return GenString(1 << 10), nil
		},
	}

	for i := 0; i < 6; i++ {
		if _, _, err := GetJSON(ThreadKey(uint64(i), 0), f); err != nil {
			t.Fatal(err)
		}
	}

	_, ok := cache[ThreadKey(0, 0)]
	if ok {
		t.Error("store not evicted")
	}
}
