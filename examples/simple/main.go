package main

import (
	"fmt"
	"net/http/cgi"
	"os"
)

func main() {
	_, err := cgi.Request()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Printf("Content-type: text/html\n\n")
	fmt.Printf("<!DOCTYPE html>\n")
	fmt.Printf("Hello World")
}
