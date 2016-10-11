package templates

import (
	"html/template"
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
)

func init() {
	TemplateRoot = "testdata"
	config.Set(config.Configs{
		Boards: []string{"a"},
	})
}

func TestBuildIndexTemplate(t *testing.T) {
	v := vars{
		Config:     template.JS("c()"),
		ConfigHash: "a",
	}
	const source = `<script>{{.Config}}</script><b>{{.ConfigHash}}</b>` +
		`{{.Navigation}}<script>{{.IsMobile}}</script>`
	tmpl, err := template.New("index").Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	_, err = buildIndexTemplate(tmpl, v, false)
	if err != nil {
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
