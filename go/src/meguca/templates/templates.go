//go:generate qtc

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"html"
	"meguca/auth"
	"meguca/config"
	"sync"
)

var (
	indexTemplates map[auth.ModerationLevel][4][]byte
	mu             sync.RWMutex
)

// Injects dynamic variables, hashes and stores compiled templates
func Compile() error {
	levels := [...]auth.ModerationLevel{
		auth.NotLoggedIn, auth.NotStaff, auth.Janitor, auth.Moderator,
		auth.BoardOwner, auth.Admin,
	}
	t := make(map[auth.ModerationLevel][4][]byte, len(levels))
	for _, pos := range levels {
		split := bytes.Split([]byte(renderIndex(pos)), []byte("$$$"))
		t[pos] = [4][]byte{split[0], split[1], split[2], split[3]}
	}

	mu.Lock()
	indexTemplates = t
	mu.Unlock()

	return nil
}

// Board renders board page HTML for noscript browsers. withIndex specifies, if
// the rendered board page should be embedded in the index page
func Board(
	b, theme string,
	page, total int,
	pos auth.ModerationLevel,
	minimal, catalog bool,
	threadHTML []byte,
) []byte {
	conf := config.GetBoardConfigs(b)
	title := html.EscapeString(fmt.Sprintf("/%s/ - %s", b, conf.Title))
	html := renderBoard(
		threadHTML,
		b, title,
		conf,
		page, total,
		pos,
		catalog,
	)

	if minimal {
		return []byte(html)
	}

	return execIndex(html, title, theme, pos)
}

// Thread renders thread page HTML for noscript browsers
func Thread(
	id uint64,
	board, title, theme string,
	abbrev, locked bool,
	pos auth.ModerationLevel,
	postHTML []byte,
) []byte {
	title = html.EscapeString(fmt.Sprintf("/%s/ - %s", board, title))
	html := renderThread(postHTML, id, board, abbrev, locked, pos)
	return execIndex(html, title, theme, pos)
}

// Execute and index template in the second pass
func execIndex(html, title, theme string, pos auth.ModerationLevel) []byte {
	mu.RLock()
	t := indexTemplates[pos]
	mu.RUnlock()

	return bytes.Join([][]byte{
		t[0],
		[]byte(title),
		t[1],
		[]byte(theme),
		t[2],
		[]byte(html),
		t[3],
	}, nil)
}

// Render index page for WASM clients
func WasmIndex(theme string) []byte {
	return []byte(renderIndexWasm(theme))
}
