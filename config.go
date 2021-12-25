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

type configResponder struct {
	Iface     string
	Filter    string
	autosense string
	instance  *pndp.ResponderObj
}

type configProxy struct {
	Iface1    string
	Iface2    string
	Filter    string
	autosense string
	instance  *pndp.ProxyObj
}

var allResponders []*configResponder
var allProxies []*configProxy

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

		if strings.HasPrefix(line, "responder") && strings.Contains(line, "{") {
			obj := configResponder{}
			filter := ""
			for {
				scanner.Scan()
				line = strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "iface") {
					obj.Iface = strings.TrimSpace(strings.TrimPrefix(line, "iface"))
				}
				if strings.HasPrefix(line, "filter") {
					filter += strings.TrimSpace(strings.TrimPrefix(line, "filter")) + ";"
					if strings.Contains(line, ";") {
						panic("Invalid config file syntax")
					}
				}
				if strings.HasPrefix(line, "autosense") {
					obj.autosense = strings.TrimSpace(strings.TrimPrefix(line, "autosense"))
				}
				if strings.HasPrefix(line, "}") {
					obj.Filter = strings.TrimSuffix(filter, ";")
					break
				}
			}

			allResponders = append(allResponders, &obj)
		} else if strings.HasPrefix(line, "proxy") && strings.Contains(line, "{") {
			obj := configProxy{}
			filter := ""
			for {
				scanner.Scan()
				line = strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "iface1") {
					obj.Iface1 = strings.TrimSpace(strings.TrimPrefix(line, "iface1"))
				}
				if strings.HasPrefix(line, "iface2") {
					obj.Iface2 = strings.TrimSpace(strings.TrimPrefix(line, "iface2"))
				}
				if strings.HasPrefix(line, "filter") {
					filter += strings.TrimSpace(strings.TrimPrefix(line, "filter")) + ";"
					if strings.Contains(line, ";") {
						panic("Invalid config file syntax")
					}
				}
				if strings.HasPrefix(line, "autosense") {
					obj.autosense = strings.TrimSpace(strings.TrimPrefix(line, "autosense"))
				}
				if strings.HasPrefix(line, "}") {
					obj.Filter = strings.TrimSuffix(filter, ";")
					break
				}
				if strings.HasPrefix(line, "}") {
					break
				}
			}
			allProxies = append(allProxies, &obj)
		} else if strings.Contains(line, "{") {
			option := strings.TrimSuffix(strings.TrimSpace(line), "{")
			option = strings.TrimSpace(option)
			if modules.ModuleList != nil {
				for i := range modules.ModuleList {
					if (*modules.ModuleList[i]).Option == option {
						var lines []string
						for {
							scanner.Scan()
							line = strings.TrimSpace(scanner.Text())
							lines = append(lines, line)
							if strings.HasPrefix(line, "}") {
								break
							}
						}
						(*modules.ModuleList[i]).ConfigCallback(lines)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for _, n := range allProxies {
		o := pndp.NewProxy(n.Iface1, n.Iface2, pndp.ParseFilter(n.Filter), n.autosense)
		n.instance = o
		o.Start()
	}

	for _, n := range allResponders {
		o := pndp.NewResponder(n.Iface, pndp.ParseFilter(n.Filter), n.autosense)
		n.instance = o
		o.Start()
	}

	WaitForSignal()

	for _, n := range allProxies {
		n.instance.Stop()
	}

	for _, n := range allResponders {
		n.instance.Stop()
	}

}
