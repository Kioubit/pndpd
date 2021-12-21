package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var GlobalDebug = false

func simpleRespond(iface string, filter []*net.IPNet) {
	requests := make(chan *NDRequest, 100)
	defer close(requests)
	go respond(iface, requests, NDP_ADV, filter)
	go listen(iface, requests, NDP_SOL)

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sigCh:
		fmt.Println("Exit")
		os.Exit(0)
	}
}

func proxy(iface1, iface2 string) {
	req_iface1_sol_iface2 := make(chan *NDRequest, 100)
	defer close(req_iface1_sol_iface2)
	go listen(iface1, req_iface1_sol_iface2, NDP_SOL)
	go respond(iface2, req_iface1_sol_iface2, NDP_SOL, nil)

	req_iface2_sol_iface1 := make(chan *NDRequest, 100)
	defer close(req_iface2_sol_iface1)
	go listen(iface2, req_iface2_sol_iface1, NDP_SOL)
	go respond(iface1, req_iface2_sol_iface1, NDP_SOL, nil)

	req_iface1_adv_iface2 := make(chan *NDRequest, 100)
	defer close(req_iface1_adv_iface2)
	go listen(iface1, req_iface1_adv_iface2, NDP_ADV)
	go respond(iface2, req_iface1_adv_iface2, NDP_ADV, nil)

	req_iface2_adv_iface1 := make(chan *NDRequest, 100)
	defer close(req_iface2_adv_iface1)
	go listen(iface2, req_iface2_adv_iface1, NDP_ADV)
	go respond(iface1, req_iface2_adv_iface1, NDP_ADV, nil)

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sigCh:
		fmt.Println("Exit")
		os.Exit(0)
	}
}

func parseFilter(f string) []*net.IPNet {
	s := strings.Split(f, ";")
	result := make([]*net.IPNet, len(s))
	for i, n := range s {
		_, cidr, err := net.ParseCIDR(n)
		if err != nil {
			panic(err)
		}
		result[i] = cidr
	}
	return result
}
