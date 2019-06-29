// Package cache provides an in-memory LRU cache for reducing the duplicate
// workload of database requests and post JSON generation
package cache

import (
	"time"

	"github.com/bakape/recache"
)

var (
	cache *recache.Cache
)

// Init cache with specified max memory usage
func Init(maxSize uint) {
	cache = recache.NewCache(recache.Options{
		MemoryLimit: maxSize,
		LRULimit:    time.Hour,
	})

	// TODO: Init frontends
}

// TODO
// // DeleteByBoard deletes all entries by the board property of Key.
// // If no entries found, this is a NOP.
// func DeleteByBoard(board string) {
// 	mu.Lock()
// 	defer mu.Unlock()

// 	for k, el := range cache {
// 		if k.Board == board {
// 			removeEntry(el)
// 		}
// 	}
// }
