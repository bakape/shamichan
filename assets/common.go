package assets

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/util"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/log"
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
	// Background video directory
	videoDir string
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

//SetVideoDir sets the background video directory to watch
func SetVideoDir(dir string) {
	rw.Lock()
	defer rw.Unlock()
	videoDir = dir
}

func generateVideoNames() {
	rw.Lock()
	defer rw.Unlock()

	// Clean up old symlinks
	files, err := ioutil.ReadDir("www/videos")

	if err != nil {
		log.Error(err)
		return
	}

	for _, f := range files {
		// Only remove symlinks
		if f.Mode()&os.ModeSymlink != 0 {
			err = os.RemoveAll(fmt.Sprintf("www/videos/%s", f.Name()))

			if err != nil {
				log.Error(err)
				return
			}
		}
	}

	// Gather video names from video directory
	videoNames = []string{"none"}
	files, err = ioutil.ReadDir(videoDir)

	if os.IsNotExist(err) {
		regenVideoDir()
	} else if err != nil {
		log.Error(err)
		return
	}

	for _, f := range files {
		name := f.Name()
		ext := filepath.Ext(name)

		if ext == ".webm" || ext == ".mp4" {
			path, err := filepath.Abs(name)

			// If fail, don't append to videoNames
			if err != nil {
				log.Error(err)
				continue
			}

			// filepath.Abs defaulted to working directory
			if !strings.Contains(path, videoDir) {
				path = fmt.Sprintf("%s%s/%s", strings.TrimSuffix(path, name), videoDir, name)
			}

			// Symlink to www/videos
			err = os.Symlink(path, fmt.Sprintf("www/videos/%s", name))

			// If fail, don't append to videoNames, ignoring existing (non-symlinked) videos
			if !os.IsExist(err) && err != nil {
				log.Error(err)
				continue
			}

			videoNames = append(videoNames, name)
		}
	}
}

func regenVideoDir() {
	log.Warn("Background videos directory not found")
	err := os.Mkdir(videoDir, 0700)

	if err != nil {
		log.Error(err)
		return
	}

	log.Info("Created background videos directory")
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

// WatchVideoDir watches the $videoDir directory for changes.
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

	// Ensure we don't regen video directory twice
	rw.Lock()
	defer rw.Unlock()
	err = watcher.Add(videoDir)

	if os.IsNotExist(err) {
		regenVideoDir()
	} else if err != nil {
		log.Error(err)
	}
}
