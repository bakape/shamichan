package imager

import (
	"image"
	"math/bits"
	"os"
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	"github.com/corona10/goimagehash"
)

var (
	_blacklistPerceptual = make(map[uint64]struct{})
	_blacklistSHA1       = make(map[string]struct{})
	_blacklistMu         sync.RWMutex
)

func propagateBlacklistedPerceptual(
	sha1 string,
	thumb image.Image,
) (err error) {
	if CheckBlacklistedSHA1(sha1) {
		return
	}

	ih, err := goimagehash.PerceptionHash(thumb)
	if err != nil {
		return
	}
	h := ih.GetHash()

	_blacklistMu.Lock()
	defer _blacklistMu.Unlock()

	for bl := range _blacklistPerceptual {
		if bits.OnesCount64(bl^h) <= 4 {
			return registerBlacklisted(h, sha1)
		}
	}
	return
}

func CheckBlacklistedSHA1(sha1 string) bool {
	_blacklistMu.RLock()
	defer _blacklistMu.RUnlock()

	_, ok := _blacklistSHA1[sha1]
	return ok
}

func LoadBlacklist() (err error) {
	_blacklistMu.Lock()
	defer _blacklistMu.Unlock()

	_blacklistPerceptual, _blacklistSHA1, err = db.LoadBlacklistedImages()
	return
}

func Blacklist(sha1 string) (err error) {
	if CheckBlacklistedSHA1(sha1) {
		return
	}

	img, err := db.GetImage(sha1)
	if err != nil {
		return
	}

	paths := assets.GetFilePaths(sha1, img.FileType, img.ThumbType)
	path := paths[1]
	if img.ThumbType == common.NoFile {
		path = paths[0]
	}

	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	thumb, _, err := image.Decode(f)
	if err != nil {
		return
	}

	h, err := goimagehash.PerceptionHash(thumb)
	if err != nil {
		return
	}

	_blacklistMu.Lock()
	defer _blacklistMu.Unlock()
	return registerBlacklisted(h.GetHash(), sha1)
}

// Write lock on _blacklistMu required
func registerBlacklisted(perceptual uint64, sha1 string) (err error) {
	_, okS := _blacklistSHA1[sha1]
	_, okP := _blacklistPerceptual[perceptual]
	if okS && okP {
		return
	}

	err = db.BlacklistImage(perceptual, sha1)
	if err != nil {
		return
	}
	_blacklistPerceptual[perceptual] = struct{}{}
	_blacklistSHA1[sha1] = struct{}{}
	return
}
