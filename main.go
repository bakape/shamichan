/*
 Simple proxy for cleaner directory structure and so we can have godoc support
*/

package main

import "gopkg.in/bakape/meguca.v2/server"

func main() {
	server.Start()
}
