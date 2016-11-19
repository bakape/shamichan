package templates

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

func TestBoardNavigation(t *testing.T) {
	_, err := BoardNavigation(lang.Packs["en_GB"])
	if err != nil {
		t.Fatal(err)
	}
}

func TestOwnedBoard(t *testing.T) {
	t.Parallel()
	conf := config.BoardTitles{
		{
			ID:    "a",
			Title: "Animu & Mango",
		},
	}
	_, err := OwnedBoard(conf, lang.Packs["en_GB"])
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := CreateBoard(lang.Packs["en_GB"]); err != nil {
		t.Fatal(err)
	}
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
