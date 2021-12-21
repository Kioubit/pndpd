package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Usage: pndpd readconfig <path to file>")
	fmt.Println("Usage: pndpd respond <interface> <optional whitelist of CIDRs separated with a semicolon>")
	fmt.Println("Usage: pndpd proxy <interface1> <interface2>")

	if len(os.Args) <= 1 {
		fmt.Println("Specify command")
		os.Exit(1)
	}
	if os.Args[1] == "respond" {
		if len(os.Args) == 4 {
			simpleRespond(os.Args[2], parseFilter(os.Args[3]))
		} else {
			simpleRespond(os.Args[2], nil)
		}
	}
	if os.Args[1] == "proxy" {
		proxy(os.Args[2], os.Args[3])
	}

	if os.Args[1] == "readConfig" {
		readConfig(os.Args[2])
	}

}
