package main

import (
	"bufio"
	"fmt"
	"os"
	"pndpd/modules"
	"pndpd/pndp"
	"strings"
)

func readConfig(dest string) {
	file, err := os.Open(dest)
	if err != nil {
		configFatalError(err, "")
	}
	var (
		currentOption string
		blockMap      map[string][]string
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line, _, _ = strings.Cut(line, "//")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if after, found := strings.CutPrefix(line, "debug"); found {
			if strings.TrimSpace(after) == "on" {
				pndp.GlobalDebug = true
				fmt.Println("DEBUG ON")
			}
			continue
		}

		if option, after, found := strings.Cut(line, "{"); found {
			if after != "" {
				configFatalError(nil, "Nothing may follow after '{'. A new line must be used")
			}
			if blockMap != nil {
				configFatalError(nil, "A new '{' block was started before the previous one was closed")
			}
			currentOption = strings.TrimSpace(option)
			blockMap = make(map[string][]string)
			continue
		}

		if before, after, found := strings.Cut(line, "}"); found {
			if after != "" || before != "" {
				configFatalError(nil, "Nothing may precede or follow '}'. A new line must be used")
			}
			if blockMap == nil {
				configFatalError(nil, "Found a '}' tag without a matching '{' tag.")
			}
			module, command := modules.GetCommand(currentOption, modules.Config)
			if module == nil {
				configFatalError(nil, "Unknown configuration block: "+currentOption)
			}
			modules.ExecuteInit(module, modules.CallbackInfo{
				CallbackType: modules.Config,
				Command:      command,
				Config:       blockMap,
			})
			blockMap = nil
			continue
		}

		if blockMap != nil {
			kv := strings.SplitN(line, " ", 2)
			if len(kv) != 2 {
				configFatalError(nil, "Key without value")
			}
			if blockMap[kv[0]] == nil {
				blockMap[kv[0]] = make([]string, 0)
			}
			blockMap[kv[0]] = append(blockMap[kv[0]], kv[1])
		}
	}
	_ = file.Close()

	if err := scanner.Err(); err != nil {
		configFatalError(err, "")
	}

	if modules.ExistsBlockingModule() {
		modules.ExecuteComplete()
		waitForSignal()
		modules.ShutdownAll()
	}
}

func configFatalError(err error, explanation string) {
	fmt.Println("Error reading config file:", explanation)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(1)
}
