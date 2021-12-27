package userInterface

import (
	"fmt"
	"pndpd/modules"
	"pndpd/pndp"
	"strings"
)

func init() {
	commands := []modules.Command{{
		CommandText:    "proxy",
		Description:    "proxy <interface1> <interface2> <optional whitelist of CIDRs separated by a semicolon applied to interface2>",
		BlockTerminate: true,
	}, {
		CommandText:    "responder",
		Description:    "responder <interface> <optional whitelist of CIDRs separated by a semicolon>",
		BlockTerminate: true,
	}, {
		CommandText:    "modules",
		Description:    "modules available - list available modules",
		BlockTerminate: false,
	}}
	modules.RegisterModule("Core", commands, initCallback, completeCallback, shutdownCallback)
}

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

func initCallback(callback modules.CallbackInfo) {
	if callback.CallbackType == modules.CommandLine {
		switch callback.Command.CommandText {
		case "proxy":
			if len(callback.Arguments) == 3 {
				allProxies = append(allProxies, &configProxy{
					Iface1:    callback.Arguments[0],
					Iface2:    callback.Arguments[1],
					Filter:    callback.Arguments[2],
					autosense: "",
					instance:  nil,
				})
			} else {
				allProxies = append(allProxies, &configProxy{
					Iface1:    callback.Arguments[0],
					Iface2:    callback.Arguments[1],
					Filter:    "",
					autosense: "",
					instance:  nil,
				})
			}
		case "responder":
			if len(callback.Arguments) == 2 {
				allResponders = append(allResponders, &configResponder{
					Iface:     callback.Arguments[0],
					Filter:    callback.Arguments[1],
					autosense: "",
					instance:  nil,
				})
			} else {
				allResponders = append(allResponders, &configResponder{
					Iface:     callback.Arguments[0],
					Filter:    "",
					autosense: "",
					instance:  nil,
				})
			}
		case "modules":
			if modules.ModuleList != nil {
				fmt.Print("Available Modules: ")
				for i := range modules.ModuleList {
					fmt.Print((*modules.ModuleList[i]).Name + " ")
				}
				fmt.Println()
			}

		}

	} else {
		switch callback.Command.CommandText {
		case "proxy":
			obj := configProxy{}
			filter := ""
			for _, n := range callback.Arguments {
				if strings.HasPrefix(n, "iface1") {
					obj.Iface1 = strings.TrimSpace(strings.TrimPrefix(n, "iface1"))
				}
				if strings.HasPrefix(n, "iface2") {
					obj.Iface2 = strings.TrimSpace(strings.TrimPrefix(n, "iface2"))
				}
				if strings.HasPrefix(n, "filter") {
					filter += strings.TrimSpace(strings.TrimPrefix(n, "filter")) + ";"
					if strings.Contains(n, ";") {
						panic("Invalid config file syntax")
					}
				}
				if strings.HasPrefix(n, "autosense") {
					obj.autosense = strings.TrimSpace(strings.TrimPrefix(n, "autosense"))
				}
			}
			obj.Filter = strings.TrimSuffix(filter, ";")
			allProxies = append(allProxies, &obj)
		case "responder":
			obj := configResponder{}
			filter := ""
			for _, n := range callback.Arguments {
				if strings.HasPrefix(n, "iface") {
					obj.Iface = strings.TrimSpace(strings.TrimPrefix(n, "iface"))
				}
				if strings.HasPrefix(n, "filter") {
					filter += strings.TrimSpace(strings.TrimPrefix(n, "filter")) + ";"
					if strings.Contains(n, ";") {
						panic("Invalid config file syntax")
					}
				}
				if strings.HasPrefix(n, "autosense") {
					obj.autosense = strings.TrimSpace(strings.TrimPrefix(n, "autosense"))
				}

			}
			obj.Filter = strings.TrimSuffix(filter, ";")
			allResponders = append(allResponders, &obj)

		}
	}
}

func completeCallback() {
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
}
func shutdownCallback() {
	for _, n := range allProxies {
		n.instance.Stop()
	}

	for _, n := range allResponders {
		n.instance.Stop()
	}
}
