package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("Specify interface")
		os.Exit(1)
	}
	iface := os.Args[1]
	requests := make(chan *NDRequest, 100)
	defer close(requests)
	go respond(iface, requests)
	listen(iface, requests)

}
