package templates

import (
	"bytes"
	"fmt"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

// Thread renders thread page HTML for noscript browsers
func Thread(ln lang.Pack, withIndex bool, t common.Thread) ([]byte, error) {
	w := new(bytes.Buffer)
	conf := config.GetBoardConfigs(t.Board)
	title := fmt.Sprintf("/%s/ - %s (#%d)", t.Board, t.Subject, t.ID)

	v := struct {
		Title, Root string
		Thread      common.Thread
		Conf        config.BoardPublic
		Lang        lang.Pack
	}{
		Root:   config.Get().RootURL,
		Title:  title,
		Thread: t,
		Conf:   conf.BoardPublic,
		Lang:   ln,
	}

	err := tmpl["thread"].Execute(w, v)
	if err != nil {
		return nil, err
	}

	if !withIndex {
		return w.Bytes(), nil
	}

	return execIndex(w, ln.ID, title)
}
