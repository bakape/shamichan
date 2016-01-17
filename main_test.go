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
	var paniced bool
	os.Args[1] = "init"
	defer func() {
		if recover() != nil {
			paniced = true
		}
	}()
	main()
	c.Assert(paniced, Equals, false)
}
