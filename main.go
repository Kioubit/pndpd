package main

import (
	"fmt"
	"os"
	"pndpd/pndp"
)

func main() {
	fmt.Println("PNDPD Version 0.4 by Kioubit")

	if len(os.Args) <= 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "respond":
		if len(os.Args) == 4 {
			pndp.SimpleRespond(os.Args[2], pndp.ParseFilter(os.Args[3]))
		} else {
			pndp.SimpleRespond(os.Args[2], nil)
		}
	case "proxy":
		pndp.Proxy(os.Args[2], os.Args[3])
	case "readconfig":
		readConfig(os.Args[2])
	default:
		printUsage()
		return
	}
	pndp.WaitForSignal()
}

func printUsage() {
	fmt.Println("Specify command")
	fmt.Println("Usage: pndpd readconfig <path to file>")
	fmt.Println("Usage: pndpd respond <interface> <optional whitelist of CIDRs separated with a semicolon>")
	fmt.Println("Usage: pndpd proxy <interface1> <interface2>")
}
