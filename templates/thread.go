package templates

import (
	"encoding/json"
	"fmt"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/lang"
)

// Thread renders thread page HTML for noscript browsers
func Thread(ln lang.Pack, withIndex bool, t common.Thread) ([]byte, error) {
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

	html := renderThread(t, postData, title, omit, int(imgOmit), ln)

	if !withIndex {
		return []byte(html), nil
	}
	return execIndex(html, title, ln.ID)
}
