package config

import (
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type Tests struct{}

var _ = Suite(&Tests{})

func (*Tests) TestLoadConfig(c *C) {
	configRoot = "test"
	def := filepath.FromSlash(configRoot + "/defaults.json")
	path := filepath.FromSlash(configRoot + "/config.json")
	standard, err := ioutil.ReadFile(def)
	c.Assert(err, IsNil)

	// Config file does not exist
	func() {
		defer func() {
			err := recover().(error)	
			c.Assert(
				os.IsNotExist(err),
				Equals,
				true,
				Commentf("expected: ENOENT\ngot: %s", err),
			)
		}()
		LoadConfig()
	}()

	c.Assert(ioutil.WriteFile(path, standard, 0600), IsNil)
	defer func() {
		c.Assert(os.Remove(path), IsNil)
	}()

	LoadConfig()
	stdConfig := Server{}
	stdConfig.Posts.Salt = "LALALALALALALALALALALALALALALALALALALALA"
	c.Assert(Config, DeepEquals, stdConfig)
	c.Assert(Hash, Equals, "eeba38176564a577")
}
