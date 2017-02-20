//go:generate qtc

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"html"
	"sync"
	"text/template"

	"meguca/common"
	"meguca/config"
	"meguca/lang"
)

var (
	indexTemplates = make(map[string]*template.Template, 6)
	mu             sync.RWMutex
)

// Compile reads template HTML from disk, injects dynamic variables,
// hashes and stores them
func Compile() error {
	t := make(map[string]*template.Template, len(lang.Packs))
	for id := range lang.Packs {
		ln := lang.Packs[id]
		firstPass := renderIndex(ln)

		var err error
		t[id], err = template.New(ln.ID).Parse(firstPass)
		if err != nil {
			return err
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
	minimal, catalog bool,
	threadHTML []byte,
) ([]byte, error) {
	boardConf := config.GetBoardConfigs(b)
	title := fmt.Sprintf("/%s/ - %s", b, boardConf.Title)
	title = html.EscapeString(title)

	html := renderBoard(threadHTML, b, title, boardConf, catalog, ln)

	if minimal {
		return []byte(html), nil
	}
	return execIndex(html, title, ln.ID)
}

// Thread renders thread page HTML for noscript browsers
func Thread(ln lang.Pack, minimal bool, postHTML []byte) ([]byte, error) {
	html := renderThread(postHTML, ln)
	if minimal {
		return []byte(html), nil
	}
	return execIndex(html, "", ln.ID)
}

// CalculateOmit returns the omitted post and image counts for a thread
func CalculateOmit(t common.Thread) (int, int) {
	// There might still be posts missing due to deletions even in complete
	// thread queries. Ensure we are actually retrieving an abbreviated thread
	// before calculating.
	if !t.Abbrev {
		return 0, 0
	}

	var (
		omit    = int(t.PostCtr) - (len(t.Posts) + 1)
		imgOmit uint32
	)
	if omit != 0 {
		imgOmit = t.ImageCtr
		if t.Image != nil {
			imgOmit--
		}
		for _, p := range t.Posts {
			if p.Image != nil {
				imgOmit--
			}
		}
	}
	return omit, int(imgOmit)
}

// Execute and index template in the second pass
func execIndex(html, title, lang string) ([]byte, error) {
	mu.RLock()
	t := indexTemplates[lang]
	mu.RUnlock()

	var w bytes.Buffer
	err := t.Execute(&w, struct {
		Title, Threads string
	}{
		Title:   title,
		Threads: html,
	})
	return w.Bytes(), err
}

func bold(s string) string {
	s = html.EscapeString(s)
	b := make([]byte, 3, len(s)+7)
	copy(b, "<b>")
	b = append(b, s...)
	b = append(b, "</b>"...)
	return string(b)
}
