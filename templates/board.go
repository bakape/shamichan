package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"sort"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/types"
)

type boardVars struct {
	IsAll, NeedImage, Captcha                bool
	Banner, Notice, Rules, Title, CaptchaKey string
	SortModes []string
	Threads                                  types.BoardThreads
	Boards                                   []config.BoardTitle
	Lang                                     lang.Pack
}

// Board renders board page HTML for noscript browsers
func Board(b string, lang lang.Pack, data types.Board) ([]byte, error) {
	w := new(bytes.Buffer)
	conf := config.Get()
	boardConf := config.GetBoardConfigs(b)
	title := fmt.Sprintf("/%s/ - %s", b, boardConf.Title)
	sort.Sort(data.Threads) // Sort by last reply time

	v := boardVars{
		IsAll:      b == "all",
		NeedImage:  !boardConf.TextOnly,
		Notice:     boardConf.Notice,
		Rules:      boardConf.Rules,
		Title:      title,
		Threads:    data.Threads,
		Lang:       lang,
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

	return execIndex(w, lang.ID, title)
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
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}
