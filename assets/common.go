package assets

import (
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/util"
)

// Export to avoid circular dependency
func init() {
	common.GetVideoNames = GetVideoNames
}

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
