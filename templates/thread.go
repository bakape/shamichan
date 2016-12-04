package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

// Thread renders thread page HTML for noscript browsers
func Thread(ln lang.Pack, withIndex bool, t common.Thread) ([]byte, error) {
	w := new(bytes.Buffer)
	conf := config.GetBoardConfigs(t.Board)
	title := fmt.Sprintf("/%s/ - %s (#%d)", t.Board, t.Subject, t.ID)

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

	v := struct {
		Title, Root      string
		Thread           common.Thread
		Conf             config.BoardPublic
		Omit, ImageOmit  int
		Lang             lang.Pack
		JSON, ConfigJSON template.JS
	}{
		Root:       config.Get().RootURL,
		Title:      title,
		Thread:     t,
		Omit:       omit,
		ImageOmit:  int(imgOmit),
		Conf:       conf.BoardPublic,
		Lang:       ln,
		JSON:       template.JS(postData),
		ConfigJSON: template.JS(conf.JSON),
	}

	if err = tmpl["thread"].Execute(w, v); err != nil {
		return nil, err
	}

	if !withIndex {
		return w.Bytes(), nil
	}

	return execIndex(w, ln.ID, title)
}
