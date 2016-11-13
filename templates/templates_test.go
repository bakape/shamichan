package templates

import (
	"testing"

	"path/filepath"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

func init() {
	TemplateRoot = "."
	isTest = true
	_, err := config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})
	if err != nil {
		panic(err)
	}

	config.Set(config.Configs{})

	lang.Dir = filepath.Join("..", "lang")
	if err := lang.Load(); err != nil {
		panic(err)
	}

	if err := ParseTemplates(); err != nil {
		panic(err)
	}
}

func TestCompileTemplates(t *testing.T) {
	config.SetClient([]byte{1}, "hash")
	(*config.Get()).Captcha = true

	if err := Compile(); err != nil {
		t.Fatal(err)
	}
}
