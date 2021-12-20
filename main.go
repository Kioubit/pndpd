package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Usage: pndpd respond <interface>")
	fmt.Println("Usage: pndpd proxy <interface1> <interface2>")

	if len(os.Args) <= 1 {
		fmt.Println("Specify command")
		os.Exit(1)
	}
	if os.Args[1] == "respond" {
		simpleRespond(os.Args[2])
	}
	if os.Args[1] == "proxy" {
		proxy(os.Args[2], os.Args[3])
	}

}

func simpleRespond(iface string) {
	requests := make(chan *NDRequest, 100)
	defer close(requests)
	go respond(iface, requests, NDP_ADV)
	go listen(iface, requests, NDP_SOL)
	select {}
	//TODO os.signal
}

func proxy(iface1, iface2 string) {
	req_iface1_sol_iface2 := make(chan *NDRequest, 100)
	defer close(req_iface1_sol_iface2)
	go listen(iface1, req_iface1_sol_iface2, NDP_SOL)
	go respond(iface2, req_iface1_sol_iface2, NDP_SOL)

	req_iface2_sol_iface1 := make(chan *NDRequest, 100)
	defer close(req_iface2_sol_iface1)
	go listen(iface2, req_iface2_sol_iface1, NDP_SOL)
	go respond(iface1, req_iface2_sol_iface1, NDP_SOL)

	req_iface1_adv_iface2 := make(chan *NDRequest, 100)
	defer close(req_iface1_adv_iface2)
	go listen(iface1, req_iface1_adv_iface2, NDP_ADV)
	go respond(iface2, req_iface1_adv_iface2, NDP_ADV)

	req_iface2_adv_iface1 := make(chan *NDRequest, 100)
	defer close(req_iface2_adv_iface1)
	go listen(iface2, req_iface2_adv_iface1, NDP_ADV)
	go respond(iface1, req_iface2_adv_iface1, NDP_ADV)

	select {}
	// TODO os.signal

}
