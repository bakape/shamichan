package assets

import (
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/log"

	"meguca/common"
	"meguca/util"
)

// Export to avoid circular dependency
func init() {
	common.GetVideoNames = GetVideoNames
}

var (
	// Generate video names on first GetVideoNames call
	once sync.Once
	// Prevents data race with generating or getting video names
	rw sync.RWMutex
	// List of video filenames
	videoNames []string
)

// File contains data and type of a file stored in memory
type File struct {
	Data       []byte
	Mime, Hash string
}

// FileStore stores board-specific files in memory
type FileStore struct {
	mu  sync.RWMutex
	m   map[string]File
	def File
}

// Set file stored for a certain board.
// If file is a zero struct, previous file is deleted.
// Technically deleting a board would leak memory, but it's so rare and little.
func (s *FileStore) Set(board string, file File) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if file.Data == nil {
		delete(s.m, board)
	} else {
		file.Hash = util.HashBuffer(file.Data)
		s.m[board] = file
	}
}

// Get returns the file specified by board. If none found, default is returned.
// file should not be mutted.
func (s *FileStore) Get(board string) (file File) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, ok := s.m[board]
	if !ok {
		return s.def
	}
	return file
}

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
	once.Do(func() {
		generateVideoNames()
	})

	rw.RLock()
	defer rw.RUnlock()
	return videoNames
}

// WatchVideoDir watches the www/videos directory for changes
func WatchVideoDir() {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Error(err)
	}

	defer watcher.Close()

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
