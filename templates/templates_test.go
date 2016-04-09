package templates

import (
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
	"html/template"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type Templates struct{}

var _ = Suite(&Templates{})

func (t *Templates) TestBoardNavigation(c *C) {
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a", "staff"}
	conf.Boards.Staff = "staff"
	conf.Boards.Psuedo = [][2]string{
		[2]string{"g", "https://google.com"},
	}
	config.Set(conf)
	html := boardNavigation()
	std := `<b id="navTop">[<a href="../a/">a</a> / <a href="../all/">all</a>` +
		` / <a href="https://google.com">g</a>]</b>`
	c.Assert(string(html), Equals, std)
}

func (t *Templates) TestBuildIndexTemplate(c *C) {
	v := vars{
		Config:     template.JS("c()"),
		ConfigHash: "a",
		Navigation: template.HTML("<hr>"),
	}
	source := `<script>{{.Config}}</script><b>{{.ConfigHash}}</b>` +
		`{{.Navigation}}<script>{{.IsMobile}}</script>`
	tmpl, err := template.New("index").Parse(source)
	c.Assert(err, IsNil)
	standard := Store{
		HTML: []byte("<script>c()</script><b>a</b><hr><script>false</script>"),
		Hash: "d99e2949415f7ec0",
	}
	res, err := buildIndexTemplate(tmpl, v, false)
	c.Assert(err, IsNil)
	c.Assert(res, DeepEquals, standard)
}

func (t *Templates) TestCompileTemplates(c *C) {
	config.SetClient([]byte{1}, "hash")
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)
	templateRoot = "test"
	standard := Store{
		HTML: []byte("<a></a>\n"),
		Hash: "eb51aca26e55050a",
	}
	defer func() {
		c.Assert(recover(), IsNil)
	}()
	resources = map[string]Store{}
	c.Assert(Compile(), IsNil)
	c.Assert(Get("index"), DeepEquals, standard)
	c.Assert(Get("mobile"), DeepEquals, standard)
}
