package main

import (
	"bufio"
	"log"
	"os"
	"pndpd/pndp"
	"strings"
)

type configResponder struct {
	Iface  string
	Filter string
}

type configProxy struct {
	Iface1 string
	Iface2 string
}

func readConfig(dest string) {
	file, err := os.Open(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "debug") {
			if strings.Contains(line, "off") {
				pndp.GlobalDebug = false
			}
			continue
		}
		if strings.HasPrefix(line, "responder") {
			obj := configResponder{}
			filter := ""
			for {
				scanner.Scan()
				line = scanner.Text()
				if strings.HasPrefix(line, "iface") {
					obj.Iface = strings.TrimSpace(strings.TrimPrefix(line, "iface"))
				}
				if strings.HasPrefix(line, "filter") {
					filter += strings.TrimSpace(strings.TrimPrefix(line, "filter")) + ";"
				}
				if strings.HasPrefix(line, "}") {
					obj.Filter = filter
					break
				}
			}
			pndp.SimpleRespond(obj.Iface, pndp.ParseFilter(obj.Filter))
		}
		if strings.HasPrefix(line, "proxy") {
			obj := configProxy{}
			for {
				scanner.Scan()
				line = scanner.Text()
				if strings.HasPrefix(line, "iface1") {
					obj.Iface1 = strings.TrimSpace(strings.TrimPrefix(line, "iface1"))
				}
				if strings.HasPrefix(line, "iface2") {
					obj.Iface2 = strings.TrimSpace(strings.TrimPrefix(line, "iface2"))
				}
				if strings.HasPrefix(line, "}") {
					break
				}
			}
			pndp.Proxy(obj.Iface1, obj.Iface2)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
