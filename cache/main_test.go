package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/test"
)

// Basic test for deadlocks
func TestConcurrency(t *testing.T) {
	Clear()
	config.Server.CacheSize = 1

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
					if _, _, _, err := GetJSONAndData(k, f); err != nil {
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

	config.Server.CacheSize = 0.005
	f := FrontEnd{
		GetCounter: func(k Key) (uint64, error) {
			return 1, nil
		},
		GetFresh: func(k Key) (interface{}, error) {
			return test.GenString(1 << 10), nil
		},
	}

	for i := 0; i < 6; i++ {
		_, _, _, err := GetJSONAndData(ThreadKey(uint64(i), 0), f)
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Second * 1) // Wait for goroutine
	mu.Lock()
	defer mu.Unlock()
	_, ok := cache[ThreadKey(0, 0)]
	if ok {
		t.Error("store not evicted")
	}
}
