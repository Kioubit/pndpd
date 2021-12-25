package example

import (
	"fmt"
	"pndpd/modules"
)

// This is an example module that is not imported by the main program
func init() {
	modules.RegisterModule("Example", "example", "example <parameter 1> <parameter 2>", commandLineRead, configRead)
}

func configRead(s []string) {
	// Prints out the contents of the config file that are relevant for this module (that are inside the example{} option)
	for _, n := range s {
		fmt.Println(n)
	}
}

func commandLineRead(s []string) {
	// Prints out the command line options given to the program if the command starts with "example"
	for _, n := range s {
		fmt.Println(n)
	}
}
