package templates

import (
	"testing"

	"github.com/bakape/meguca/lang"
)

func TestBoardNavigation(t *testing.T) {
	_, err := BoardNavigation(lang.Packs["en_GB"])
	if err != nil {
		t.Fatal(err)
	}
}
