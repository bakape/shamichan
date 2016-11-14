package templates

import (
	"path/filepath"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/util"
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
	fns := []func() error{lang.Load, ParseTemplates, Compile}
	if err := util.Waterfall(fns); err != nil {
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
