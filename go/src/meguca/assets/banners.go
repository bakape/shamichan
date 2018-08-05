package assets

import (
	"math/rand"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/util"
	"sync"
	"time"
)

var (
	// Banners by board stored in memory
	Banners = BannerStore{
		m: make(map[string][]File, 64),
	}
)

func init() {
	rand.Seed(time.Now().Unix())
}

// BannerStore stores multiple files by board in memory
type BannerStore struct {
	mu sync.RWMutex
	m  map[string][]File
}

// Set files stored for a certain board.
// Technically deleting a board would leak memory, but it's so rare and little.
func (s *BannerStore) Set(board string, files []File) {
	s.mu.Lock()
	for i := range files {
		files[i].Hash = util.HashBuffer(files[i].Data)
	}
	s.m[board] = files
	s.mu.Unlock()

	// Patch global configurations
	if auth.IsNonMetaBoard(board) { // In case of some kind of DB data race
		c := config.GetBoardConfigs(board).BoardConfigs
		c.Banners = s.FileTypes(board)
		config.SetBoardConfigs(c)
	}
}

// Get returns the banner specified by board and ID. If none found, ok == false.
// file should not be mutted.
func (s *BannerStore) Get(board string, id int) (file File, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files := s.m[board]
	if id < 0 || id >= len(files) {
		return
	}
	return files[id], true
}

// Random returns a random banner for the board. If none found, ok == false.
func (s *BannerStore) Random(board string) (int, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files := s.m[board]
	if len(files) == 0 {
		return 0, "", false
	}
	i := rand.Intn(len(files))
	return i, files[i].Mime, true
}

// FileTypes returns file types of banners for a specific board
func (s *BannerStore) FileTypes(board string) []uint16 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]uint16, len(s.m[board]))
	for i, f := range s.m[board] {
		var t uint8
		switch f.Mime {
		case "image/jpeg":
			t = common.JPEG
		case "image/png":
			t = common.PNG
		case "image/gif":
			t = common.GIF
		case "video/webm":
			t = common.WEBM
		}
		types[i] = uint16(t)
	}
	return types
}
