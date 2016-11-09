package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"sort"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
)

type noscriptVars struct {
	Title, DefaultCSS string
	Threads           template.HTML
	Boards            []string
}

type boardVars struct {
	IsAll, NeedImage                  bool
	Banner, Notice, Title, CaptchaKey string
	Threads                           types.BoardThreads
	Boards                            []config.BoardTitle
}

type threadVars struct {
	Notice, Title string
	Thread        *types.Thread
}

// Board renders board page HTML for noscript browsers
func Board(b string, data *types.Board) ([]byte, error) {
	w := new(bytes.Buffer)
	conf := config.GetBoardConfigs(b)
	title := fmt.Sprintf("/%s/ - %s", b, conf.Title)
	sort.Sort(data.Threads) // Sort by last reply time

	v := boardVars{
		IsAll:     b == "all",
		NeedImage: !conf.TextOnly,
		Notice:    conf.Notice,
		Title:     title,
		Threads:   data.Threads,
	}
	if len(conf.Banners) != 0 {
		v.Banner = conf.Banners[rand.Intn(len(conf.Banners))]
	}
	gConf := config.Get()
	if gConf.Captcha {
		v.CaptchaKey = gConf.CaptchaPublicKey
	}
	if v.IsAll {
		v.Boards = config.GetBoardTitles()
	}

	err := tmpl["board"].Execute(w, v)
	if err != nil {
		return nil, err
	}

	return renderNoscriptIndex(w.Bytes(), title)
}

// Common part of both thread and board noscript pages
func renderNoscriptIndex(data []byte, title string) ([]byte, error) {
	w := new(bytes.Buffer)
	boards := config.GetBoards()
	sort.Strings(boards)

	err := tmpl["noscript"].Execute(w, noscriptVars{
		Threads:    template.HTML(data),
		Boards:     append([]string{"all"}, boards...),
		DefaultCSS: config.Get().DefaultCSS,
		Title:      title,
	})
	return w.Bytes(), err
}

// Thread renders thread page HTML for noscript browsers
func Thread(t *types.Thread) ([]byte, error) {
	w := new(bytes.Buffer)
	conf := config.GetBoardConfigs(t.Board)
	title := fmt.Sprintf("/%s/ - %s (#%d)", t.Board, t.Subject, t.ID)

	v := threadVars{
		Notice: conf.Notice,
		Title:  title,
		Thread: t,
	}

	err := tmpl["thread"].Execute(w, v)
	if err != nil {
		return nil, err
	}

	return renderNoscriptIndex(w.Bytes(), title)
}
