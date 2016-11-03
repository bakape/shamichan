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
	Hats, IsAll           bool
	Banner, Notice, Title string
	Threads               types.BoardThreads
}

// Board returns board page HTML for noscript browsers
func Board(b string, data *types.Board) ([]byte, error) {
	w := new(bytes.Buffer)
	conf := config.GetBoardConfigs(b)
	title := fmt.Sprintf("/%s/ - %s", b, conf.Title)

	board, err := renderBoard(data, conf.BoardConfigs, title)
	if err != nil {
		return nil, err
	}

	err = tmpl["noscript"].ExecuteTemplate(w, "noscript.html", noscriptVars{
		Threads:    template.HTML(board),
		Boards:     append([]string{"all"}, config.GetBoards()...),
		DefaultCSS: config.Get().DefaultCSS,
		Title:      title,
	})

	return w.Bytes(), err
}

func renderBoard(data *types.Board, conf config.BoardConfigs, title string) (
	[]byte, error,
) {
	w := new(bytes.Buffer)
	sort.Sort(data.Threads) // Sort by last reply time
	v := boardVars{
		IsAll:   conf.ID == "all",
		Hats:    config.Get().Hats,
		Notice:  conf.Notice,
		Title:   title,
		Threads: data.Threads,
	}
	if len(conf.Banners) != 0 {
		v.Banner = conf.Banners[rand.Intn(len(conf.Banners))]
	}

	err := tmpl["board"].ExecuteTemplate(w, "board.html", v)
	return w.Bytes(), err
}

// Returns the HTTP path to the thumbnail of an image
func thumbPath(img *types.Image) string {
	var ext string
	if img.FileType == types.JPEG {
		ext = "jpg"
	} else {
		ext = "png"
	}
	return fmt.Sprintf("/images/thumb/%s.%s", img.SHA1, ext)
}
