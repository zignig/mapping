package main

import (
	"fmt"
)

func main() {
	fmt.Println("Start Mapping application")
	w := NewWebServer()
	w.Run()
}
