package main

import (
	"fmt"
	"os"
	"os/signal"
	"pndpd/modules"
	_ "pndpd/modules/userInterface"
	"syscall"
)

// waitForSignal Waits (blocking) for the program to be interrupted by the OS
func waitForSignal() {
	var sigCh = make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	close(sigCh)
}

func main() {
	fmt.Println("PNDPD Version 1.2.1 - Kioubit 2021")

	if len(os.Args) <= 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "config":
		readConfig(os.Args[2])
	default:
		module, command := modules.GetCommand(os.Args[1])
		if module != nil {
			modules.ExecuteInit(module, modules.CallbackInfo{
				CallbackType: modules.CommandLine,
				Command:      command,
				Arguments:    os.Args[2:],
			})
			if modules.ExistsBlockingModule() {
				modules.ExecuteComplete()
				waitForSignal()
				modules.ShutdownAll()
			}
		} else {
			printUsage()
		}
	}

}

func printUsage() {
	fmt.Println("More options and additional documentation in the example config file")
	fmt.Println("Usage:")
	fmt.Println("pndpd config <path to file>")
	for i := range modules.ModuleList {
		for d := range (*modules.ModuleList[i]).Commands {
			fmt.Println("pndpd", (*modules.ModuleList[i]).Commands[d].Description)
		}
	}
}
