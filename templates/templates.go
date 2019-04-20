//go:generate qtc --ext html

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"sync"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
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

// Board writes board HTML to w
func Board(w io.Writer, b, theme string, page, total int,
	pos common.ModerationLevel, minimal, catalog bool, threadHTML []byte,
) {
	conf := config.GetBoardConfigs(b)
	title := html.EscapeString(fmt.Sprintf("/%s/ - %s", b, conf.Title))
	write := func(w io.Writer) {
		writerenderBoard(w, threadHTML, b, title, conf, page, total, pos,
			catalog)
	}

	if minimal {
		write(w)
	} else {
		execIndex(w, title, theme, pos, write)
	}
}

// Thread writes thread page HTML
func Thread(w io.Writer, id uint64, board, title, theme string, abbrev,
	locked bool, pos common.ModerationLevel, postHTML []byte,
) {
	title = html.EscapeString(fmt.Sprintf("/%s/ - %s", board, title))
	execIndex(w, title, theme, pos, func(w io.Writer) {
		writerenderThread(w, postHTML, id, board, abbrev, locked, pos)
	})
}

// Execute and index template in the second pass
func execIndex(w io.Writer, title, theme string, pos common.ModerationLevel,
	fn func(w io.Writer),
) {
	mu.RLock()
	t := indexTemplates[pos]
	mu.RUnlock()

	w.Write(t[0])
	w.Write([]byte(title))
	w.Write(t[1])
	w.Write([]byte(theme))
	w.Write(t[2])
	fn(w)
	w.Write(t[3])
}
