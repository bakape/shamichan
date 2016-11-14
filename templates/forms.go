// Renders various HTML forms

package templates

import (
	"bytes"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

// BoardNavigation renders a board selection and search form
func BoardNavigation(ln lang.Pack) ([]byte, error) {
	var w bytes.Buffer
	err := tmpl["boardNavigation"].Execute(&w, struct {
		Boards config.BoardTitles
		Lang   lang.Pack
	}{
		Boards: config.GetBoardTitles(),
		Lang:   ln,
	})
	return w.Bytes(), err
}
