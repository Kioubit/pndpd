package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"pndpd/modules"
	"pndpd/pndp"
	"strings"
)

func readConfig(dest string) {
	file, err := os.Open(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "//") || strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "debug") {
			if strings.Contains(line, "on") {
				pndp.GlobalDebug = true
				fmt.Println("DEBUG ON")
			}
			continue
		}

		if strings.HasSuffix(line, "{") {
			option := strings.TrimSuffix(strings.TrimSpace(line), "{")
			option = strings.TrimSpace(option)
			module, command := modules.GetCommand(option)
			var lines = make([]string, 0)
			if module != nil {
				for {
					if !scanner.Scan() {
						break
					}
					line := strings.TrimSpace(scanner.Text())
					if strings.Contains(line, "}") {
						break
					}

					lines = append(lines, line)
				}
				modules.ExecuteInit(module, modules.CallbackInfo{
					CallbackType: modules.Config,
					Command:      command,
					Arguments:    lines,
				})
			}
		}

	}
	if modules.ExistsBlockingModule() {
		modules.ExecuteComplete()
		waitForSignal()
		modules.ShutdownAll()
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

}
