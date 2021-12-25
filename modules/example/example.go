package example

import (
	"fmt"
	"pndpd/modules"
)

// This is an example module that is not imported by the main program
func init() {
	option := []modules.Option{{
		Option:      "command1",
		Description: "This is the usage description for command1",
	}, {
		Option:      "command2",
		Description: "This is the usage description for command2",
	},
	}
	modules.RegisterModule("Example", option, callback)
}

func callback(callback modules.Callback) {
	if callback.CallbackType == modules.CommandLine {
		// The command registered by the module has been run in the commandline
		// "arguments" contains the os.Args[] passed to the program after the command registered by this module
		fmt.Println("Command: ", callback.Option)
		fmt.Println(callback.Arguments)

	} else {
		// The command registered by the module was found in the config file
		// "arguments" contains the lines between the curly braces
		fmt.Println("Command: ", callback.Option)
		fmt.Println(callback.Arguments)
	}
	fmt.Println()
}
