package server

import (
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
)

type Config struct{}

var _ = Suite(&Config{})

func (cf *Config) TestLoadConfig(c *C) {
	configRoot = "./test"
	def := configRoot + "/defaults.json"
	path := configRoot + "/config.json"
	standard, err := ioutil.ReadFile(def)
	c.Assert(err, IsNil)
	err = os.Remove(path)
	if !os.IsNotExist(err) {
		c.Assert(err, IsNil)
	}

	// Config file does not exist
	loadConfig()
	file, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(file, DeepEquals, standard)
	removeFile(path, c)

	// Invalid file
	err = os.Mkdir(path, 0600)
	c.Assert(err, IsNil)
	defer func() {
		c.Assert(recover() != nil, Equals, true)
		removeFile(path, c)
	}()
	loadConfig()
}

func removeFile(path string, c *C) {
	err := os.Remove(path)
	c.Assert(err, IsNil)
}
