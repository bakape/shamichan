//go:generate qtc --ext html

// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"text/template"

	"html"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
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
	withIndex bool,
	data common.Board,
) ([]byte, error) {
	boardConf := config.GetBoardConfigs(b)
	title := fmt.Sprintf("/%s/ - %s", b, boardConf.Title)
	title = html.EscapeString(title)

	html := renderBoard(data, b, title, boardConf, ln)

	if !withIndex {
		return []byte(html), nil
	}
	return execIndex(html, title, ln.ID)
}

// Thread renders thread page HTML for noscript browsers
func Thread(ln lang.Pack, withIndex bool, t common.Thread) ([]byte, error) {
	title := fmt.Sprintf("/%s/ - %s (#%d)", t.Board, t.Subject, t.ID)
	title = html.EscapeString(title)

	postData, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	// Calculate omitted posts and images
	var (
		omit    = int(t.PostCtr) - len(t.Posts)
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

	html := renderThread(t, postData, title, omit, int(imgOmit), ln)

	if !withIndex {
		return []byte(html), nil
	}
	return execIndex(html, title, ln.ID)
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
