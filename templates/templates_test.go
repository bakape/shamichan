package templates

import (
	"html/template"
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
)

func init() {
	TemplateRoot = "."
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

func TestBuildIndexTemplate(t *testing.T) {
	v := vars{
		Config:     template.JS("c()"),
		ConfigHash: "a",
	}
	if _, err := buildIndexTemplate(v, false); err != nil {
		t.Fatal(err)
	}
}

func TestCompileTemplates(t *testing.T) {
	config.SetClient([]byte{1}, "hash")
	Set("index", Store{})
	Set("mobile", Store{})

	if err := Compile(); err != nil {
		t.Fatal(err)
	}
	for _, k := range [...]string{"index", "mobile"} {
		AssertDeepEquals(t, Get(k), resources[k])
	}
}
