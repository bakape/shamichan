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

func (*Templates) TestBoardNavigation(c *C) {
	html := boardNavigation()
	std := `<b id="navTop">[<a href="../a/">a</a> / <a href="../all/">all</a>` +
		`]</b>`
	c.Assert(string(html), Equals, std)
}

func (*Templates) TestBuildIndexTemplate(c *C) {
	v := vars{
		Config:     template.JS("c()"),
		ConfigHash: "a",
		Navigation: template.HTML("<hr>"),
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
	TemplateRoot = "test"
	defer func() {
		c.Assert(recover(), IsNil)
	}()
	resources = map[string]Store{}
	c.Assert(Compile(), IsNil)
	c.Assert(Get("index"), DeepEquals, resources["index"])
	c.Assert(Get("mobile"), DeepEquals, resources["mobile"])
}
