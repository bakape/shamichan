package templates

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

func TestBoardNavigation(t *testing.T) {
	BoardNavigation(lang.Packs["en_GB"].UI)
}

func TestOwnedBoard(t *testing.T) {
	t.Parallel()
	conf := config.BoardTitles{
		{
			ID:    "a",
			Title: "Animu & Mango",
		},
	}
	OwnedBoard(conf, lang.Packs["en_GB"].UI)
}

func TestConfigureBoard(t *testing.T) {
	_, err := ConfigureBoard(
		config.AllBoardConfigs.BoardConfigs,
		lang.Packs["en_GB"],
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateBoard(t *testing.T) {
	t.Parallel()
	CreateBoard(lang.Packs["en_GB"])
}

func TestConfigureServer(t *testing.T) {
	t.Parallel()
	_, err := ConfigureServer(
		config.Defaults,
		lang.Packs["en_GB"],
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChangePassword(t *testing.T) {
	t.Parallel()
	_, err := ChangePassword(lang.Packs["en_GB"])
	if err != nil {
		t.Fatal(err)
	}
}
