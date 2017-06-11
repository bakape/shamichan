//go:generate qtc

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"html"
	"meguca/auth"
	"meguca/config"
	"meguca/lang"
	"sync"
)

var (
	indexTemplates = make(map[string]map[auth.ModerationLevel][3][]byte, 7)
	mu             sync.RWMutex
)

// Compile reads template HTML from disk, injects dynamic variables,
// hashes and stores them
func Compile() error {
	t := make(map[string]map[auth.ModerationLevel][3][]byte, len(lang.Packs))
	levels := [...]auth.ModerationLevel{
		auth.NotLoggedIn, auth.NotStaff, auth.Janitor, auth.Moderator,
		auth.BoardOwner, auth.Admin,
	}
	for id := range lang.Packs {
		for _, pos := range levels {
			firstPass := renderIndex(pos, lang.Packs[id])
			if _, ok := t[id]; !ok {
				t[id] = make(map[auth.ModerationLevel][3][]byte, len(levels))
			}
			split := bytes.Split([]byte(firstPass), []byte("$$$"))
			t[id][pos] = [3][]byte{split[0], split[1], split[2]}
		}
	}

	mu.Lock()
	indexTemplates = t
	mu.Unlock()

	return nil
}

// Board renders board page HTML for noscript browsers. withIndex specifies, if
// the rendered board page should be embedded in the index page
func Board(
	b string,
	ln lang.Pack,
	page, total int,
	pos auth.ModerationLevel,
	minimal, catalog bool,
	threadHTML []byte,
) []byte {
	boardConf := config.GetBoardConfigs(b)
	title := html.EscapeString(fmt.Sprintf("/%s/ - %s", b, boardConf.Title))
	html := renderBoard(
		threadHTML,
		b, title,
		boardConf,
		page, total,
		pos,
		catalog,
		ln,
	)

	if minimal {
		return []byte(html)
	}
	return execIndex(html, title, ln.ID, pos)
}

// Thread renders thread page HTML for noscript browsers
func Thread(
	ln lang.Pack,
	id uint64,
	board string,
	pos auth.ModerationLevel,
	postHTML []byte,
) []byte {
	return execIndex(renderThread(postHTML, id, board, pos, ln), "", ln.ID, pos)
}

// Execute and index template in the second pass
func execIndex(html, title, lang string, pos auth.ModerationLevel) []byte {
	mu.RLock()
	t := indexTemplates[lang][pos]
	mu.RUnlock()

	return bytes.Join([][]byte{
		t[0],
		[]byte(title),
		t[1],
		[]byte(html),
		t[2],
	}, nil)
}
