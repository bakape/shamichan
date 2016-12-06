package templates

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

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

	html := renderBoard(data, b, title, boardConf, ln)

	if !withIndex {
		return []byte(html), nil
	}
	return execIndex(html, title, ln.ID)
}

// Execute and index template in the second pass
func execIndex(html, title, lang string) ([]byte, error) {
	var w bytes.Buffer

	mu.RLock()
	t := indexTemplates[lang]
	mu.RUnlock()

	err := t.Execute(&w, secondPassVars{
		Title:   title,
		Threads: template.HTML(html),
	})
	return w.Bytes(), err
}
