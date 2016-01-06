/*
 Simple proxy for cleaner directory structure and so we can have godoc support
*/

package main

import "./server"

func main() {
	server.Start()
}
