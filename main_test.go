// Simple test, to see if the server starts

package main

import (
	. "gopkg.in/check.v1"
	"os"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type Main struct{}

var _ = Suite(&Main{})

func (m *Main) TestServerStart(c *C) {
	os.Args = []string{os.Args[0], "init"}
	defer c.Assert(recover(), IsNil)
	main()
}
