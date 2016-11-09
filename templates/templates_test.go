package templates

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
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
	if err := ParseTemplates(); err != nil {
		panic(err)
	}
}

func TestCompileTemplates(t *testing.T) {
	config.SetClient([]byte{1}, "hash")
	Set("index", Store{})

	if err := Compile(); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, Get("index"), resources["index"])
}
