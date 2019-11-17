package assets

import (
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/log"
)

var (
	// Generate video names on first GetVideoNames call
	once sync.Once
	// Prevents data race with generating or getting video names
	rw sync.RWMutex
	// List of video filenames
	videoNames []string
)

func generateVideoNames() {
	rw.Lock()
	defer rw.Unlock()

	videoNames = []string{"none"}
	files, err := ioutil.ReadDir("www/videos")

	if err != nil {
		log.Error(err)
		return
	}

	for _, f := range files {
		name := f.Name()
		ext := filepath.Ext(name)

		if ext == ".webm" || ext == ".mp4" {
			videoNames = append(videoNames, name)
		}
	}
}

// GetVideoNames fetches videoNames behind a mutex
func GetVideoNames() []string {
	if common.IsTest {
		return []string{}
	}

	once.Do(func() {
		generateVideoNames()
	})

	rw.RLock()
	defer rw.RUnlock()
	return videoNames
}

// WatchVideoDir watches the www/videos directory for changes.
func WatchVideoDir() {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Error(err)
	}

	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}

				generateVideoNames()
				err := common.Recompile()

				if err != nil {
					log.Error(err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.Error("fsnotify error: ", err)
			}
		}
	}()

	err = watcher.Add("www/videos")

	if err != nil {
		log.Error(err)
	}
}
