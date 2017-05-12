//go:generate qtc

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"html"
	"meguca/config"
	"meguca/lang"
	"sync"
)

var (
	indexTemplates = make(map[string][3][]byte, 7)
	mu             sync.RWMutex
)

// Compile reads template HTML from disk, injects dynamic variables,
// hashes and stores them
func Compile() error {
	t := make(map[string][3][]byte, len(lang.Packs))
	for id := range lang.Packs {
		ln := lang.Packs[id]
		firstPass := renderIndex(ln)

		split := bytes.Split([]byte(firstPass), []byte("$$$"))
		t[id] = [3][]byte{split[0], split[1], split[2]}
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
	minimal, catalog bool,
	threadHTML []byte,
) []byte {
	boardConf := config.GetBoardConfigs(b)
	title := fmt.Sprintf("/%s/ - %s", b, boardConf.Title)
	title = html.EscapeString(title)

	html := renderBoard(threadHTML, b, title, boardConf, catalog, ln)

	if minimal {
		return []byte(html)
	}
	return execIndex(html, title, ln.ID)
}

// Thread renders thread page HTML for noscript browsers
func Thread(ln lang.Pack, id uint64, board string, minimal bool, postHTML []byte) []byte {
	html := renderThread(postHTML, id, board, ln)
	if minimal {
		return []byte(html)
	}
	return execIndex(html, "", ln.ID)
}

// Execute and index template in the second pass
func execIndex(html, title, lang string) []byte {
	mu.RLock()
	t := indexTemplates[lang]
	mu.RUnlock()

	return bytes.Join([][]byte{
		t[0],
		[]byte(title),
		t[1],
		[]byte(html),
		t[2],
	}, nil)
}
