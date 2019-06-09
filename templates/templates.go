//go:generate qtc --ext html

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/util"
)

// Export to avoid circular dependency
func init() {
	common.Recompile = Recompile
}

var (
	indexTemplates map[common.ModerationLevel][4][]byte
	mu             sync.RWMutex
)

// Compile injects dynamic variables, hashes and stores compiled templates
func Compile() error {
	levels := [...]common.ModerationLevel{
		common.NotLoggedIn, common.NotStaff, common.Janitor, common.Moderator,
		common.BoardOwner, common.Admin,
	}

	t := make(map[common.ModerationLevel][4][]byte, len(levels))

	for _, pos := range levels {
		split := bytes.Split([]byte(renderIndex(pos)), []byte("$$$"))
		t[pos] = [4][]byte{split[0], split[1], split[2], split[3]}
	}

	mu.Lock()
	indexTemplates = t
	mu.Unlock()

	return nil
}

// Recompile templates
func Recompile() error {
	if err := Compile(); err != nil {
		return util.WrapError("recompiling templates", err)
	}

	return nil
}
