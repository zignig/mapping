package main

import (
	"fmt"
)

var (
	rootDir string
	version string
	debug bool
	build_type = ""
)

func main() {
	debug = build_type=="debug"
	fmt.Println("Start Mapping application, version:", version)
	w := NewWebServer()
	w.Run()
}
