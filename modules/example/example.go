package example

import (
	"fmt"
	"pndpd/modules"
)

// This is an example module that is not imported by the main program
func init() {
	commands := []modules.Command{{
		CommandText:    "command1",
		Description:    "This is the usage description for command1",
		BlockTerminate: true,
	}, {
		CommandText:    "command2",
		Description:    "This is the usage description for command2",
		BlockTerminate: false,
	},
	}
	modules.RegisterModule("Example", commands, initCallback, completeCallback, shutdownCallback)
}

func initCallback(callback modules.CallbackInfo) {
	if callback.CallbackType == modules.CommandLine {
		// The command registered by the module has been run in the commandline
		// "arguments" contains the os.Args[] passed to the program after the command registered by this module
		fmt.Println("Command: ", callback.Command.CommandText)
		fmt.Println(callback.Arguments)

	} else {
		// The command registered by the module was found in the config file
		// "arguments" contains the lines between the curly braces
		fmt.Println("Command: ", callback.Command.CommandText)
		fmt.Println(callback.Arguments)
	}
	fmt.Println()
}

func completeCallback() {
	//Called after the program has passed all options by calls to initCallback()
}

func shutdownCallback() {
	fmt.Println("Terminate all work")
}
