package main

import (
	"fmt"
	"os"
	"os/signal"
	"pndpd/modules"
	"pndpd/pndp"
	"syscall"
)

// WaitForSignal Waits (blocking) for the program to be interrupted by the OS
func WaitForSignal() {
	var sigCh = make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	close(sigCh)
}

func main() {
	fmt.Println("PNDPD Version 1.0 - Kioubit 2021")
	if modules.ModuleList != nil {
		fmt.Print("Loaded Modules: ")
		for i := range modules.ModuleList {
			fmt.Print((*modules.ModuleList[i]).Name + " ")
		}
		fmt.Println()
	}

	if len(os.Args) <= 2 {
		printUsage()
		return
	}
	switch os.Args[1] {
	case "respond":
		var r *pndp.ResponderObj
		if len(os.Args) == 4 {
			r = pndp.NewResponder(os.Args[2], pndp.ParseFilter(os.Args[3]), "")
			r.Start()
		} else {
			r = pndp.NewResponder(os.Args[2], nil, "")
			r.Start()
		}
		WaitForSignal()
		r.Stop()
	case "proxy":
		var p *pndp.ProxyObj
		if len(os.Args) == 5 {
			p = pndp.NewProxy(os.Args[2], os.Args[3], pndp.ParseFilter(os.Args[4]), "")
		} else {
			p = pndp.NewProxy(os.Args[2], os.Args[3], nil, "")
		}
		WaitForSignal()
		p.Stop()
	case "config":
		readConfig(os.Args[2])
	default:
		for i := range modules.ModuleList {
			if (*modules.ModuleList[i]).Option == os.Args[1] {
				(*modules.ModuleList[i]).CommandLineCallback(os.Args)
				return
			}
		}
		printUsage()
		return
	}

}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("pndpd config <path to file>")
	fmt.Println("pndpd respond <interface> <optional whitelist of CIDRs separated by a semicolon>")
	fmt.Println("pndpd proxy <interface1> <interface2> <optional whitelist of CIDRs separated by a semicolon applied to interface2>")
	fmt.Println("More options and additional documentation in the example config file")
}
