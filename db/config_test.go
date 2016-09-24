package db

import (
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/templates"
	r "github.com/dancannon/gorethink"

	. "gopkg.in/check.v1"
)

func init() {
	isTest = true
}

func (*DBSuite) TestLoadConfigs(c *C) {
	isTest = false
	defer func() {
		isTest = false
	}()
	templates.TemplateRoot = "testdata"
	sample := ConfigDocument{
		Document{"config"},
		config.Defaults,
	}
	c.Assert(Write(r.Table("main").Insert(sample)), IsNil)

	// Intiial configs
	c.Assert(loadConfigs(), IsNil)
	c.Assert(config.Get(), DeepEquals, &config.Defaults)

	// Reload on update
	boards := []string{"a", "b", "c"}
	q := GetMain("config").
		Update(map[string][]string{
			"boards": boards,
		})
	c.Assert(Write(q), IsNil) // Race condition
	time.Sleep(time.Second)
	std := config.Defaults
	std.Boards = boards
	c.Assert(config.Get(), DeepEquals, &std)
}
