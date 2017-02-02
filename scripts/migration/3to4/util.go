package main

import (
	"fmt"
	"os"

	"github.com/dancannon/gorethink"
)

// One writes the query result into the target pointer or throws an error
func One(query gorethink.Term, res interface{}) error {
	c, err := query.Run(rSession)
	if err != nil {
		return err
	}
	return c.One(res)
}

// All writes all responses into target pointer to slice or returns error
func All(query gorethink.Term, res interface{}) error {
	c, err := query.Run(rSession)
	if err != nil {
		return err
	}
	return c.All(res)
}

func printProgress(header string, done, total int) {
	done++
	fmt.Fprintf(
		os.Stdout,
		"\r%s: %d / %d - %.2f%%",
		header,
		done, total,
		float32(done)/float32(total)*100,
	)
}
