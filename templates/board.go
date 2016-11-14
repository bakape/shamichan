package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/types"
)

// Board renders board page HTML for noscript browsers. withIndex specifies, if
// the rendered board page should be embedded in the index page
func Board(
	b string,
	ln lang.Pack,
	withIndex bool,
	data types.Board,
) ([]byte, error) {
	w := new(bytes.Buffer)
	conf := config.Get()
	boardConf := config.GetBoardConfigs(b)
	title := fmt.Sprintf("/%s/ - %s", b, boardConf.Title)

	v := struct {
		IsAll, NeedImage, Captcha                bool
		Banner, Notice, Rules, Title, CaptchaKey string
		SortModes                                []string
		Threads                                  types.BoardThreads
		Boards                                   config.BoardTitles
		Lang                                     lang.Pack
	}{
		IsAll:      b == "all",
		NeedImage:  !boardConf.TextOnly,
		Notice:     boardConf.Notice,
		Rules:      boardConf.Rules,
		Title:      title,
		Threads:    data.Threads,
		Lang:       ln,
		Captcha:    conf.Captcha,
		CaptchaKey: conf.CaptchaPublicKey,
	}
	if len(boardConf.Banners) != 0 {
		v.Banner = boardConf.Banners[rand.Intn(len(boardConf.Banners))]
	}
	if v.IsAll {
		v.Boards = config.GetBoardTitles()
	}

	err := tmpl["board"].Execute(w, v)
	if err != nil {
		return nil, err
	}

	if !withIndex {
		return w.Bytes(), nil
	}

	return execIndex(w, ln.ID, title)
}

// Execute and index template in the second pass
func execIndex(w *bytes.Buffer, lang, title string) ([]byte, error) {
	html := w.String()
	w.Reset()

	mu.RLock()
	t := indexTemplates[lang]
	mu.RUnlock()

	err := t.Execute(w, secondPassVars{
		Title:   title,
		Threads: template.HTML(html),
	})
	return w.Bytes(), err
}
