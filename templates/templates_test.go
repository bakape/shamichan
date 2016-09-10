package templates

import (
	"html/template"
	"testing"

	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type Templates struct{}

var _ = Suite(&Templates{})

func (*Templates) SetUpTest(c *C) {
	config.Set(config.Configs{
		Boards: []string{"a"},
	})
}

func (*Templates) TestBuildIndexTemplate(c *C) {
	v := vars{
		Config:     template.JS("c()"),
		ConfigHash: "a",
	}
	source := `<script>{{.Config}}</script><b>{{.ConfigHash}}</b>` +
		`{{.Navigation}}<script>{{.IsMobile}}</script>`
	tmpl, err := template.New("index").Parse(source)
	c.Assert(err, IsNil)
	_, err = buildIndexTemplate(tmpl, v, false)
	c.Assert(err, IsNil)
}

func (*Templates) TestCompileTemplates(c *C) {
	config.SetClient([]byte{1}, "hash")
	TemplateRoot = "testdata"
	defer func() {
		c.Assert(recover(), IsNil)
	}()
	resources = map[string]Store{}
	c.Assert(Compile(), IsNil)
	c.Assert(Get("index"), DeepEquals, resources["index"])
	c.Assert(Get("mobile"), DeepEquals, resources["mobile"])
}
