package main

import (
	"fmt"
)

func main() {
	fmt.Printf("Content-type: text/html\n\n")
	fmt.Printf("<!DOCTYPE html>\n")
	for i := 0; ; i++ {
		fmt.Printf("%d\n", i)
	}
}
