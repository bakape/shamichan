// Simple proxy for cleaner directory structure and so we can have godoc support

package main

import "github.com/bakape/meguca/server"

func main() {
	err := server.Start()
	if err != nil {
		panic(err)
	}
}
