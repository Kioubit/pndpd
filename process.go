package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"
)

var GlobalDebug = false

// Items needed for graceful shutdown
var stop = make(chan struct{})
var stopWg sync.WaitGroup
var sigCh = make(chan os.Signal)

func waitForSignal() {
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	fmt.Println("Shutting down...")
	close(stop)
	if wgWaitTimout(&stopWg, 10*time.Second) {
		fmt.Println("Done")
	} else {
		fmt.Println("Aborting shutdown, since it is taking too long")
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	}

	os.Exit(0)
}

func wgWaitTimout(wg *sync.WaitGroup, timeout time.Duration) bool {
	t := make(chan struct{})
	go func() {
		defer close(t)
		wg.Wait()
	}()
	select {
	case <-t:
		return true
	case <-time.After(timeout):
		return false
	}
}

func simpleRespond(iface string, filter []*net.IPNet) {
	defer stopWg.Done()
	stopWg.Add(3) // This function, 2x goroutines
	requests := make(chan *NDRequest, 100)
	defer close(requests)
	go respond(iface, requests, NDP_ADV, filter)
	go listen(iface, requests, NDP_SOL)
	<-stop
}

func proxy(iface1, iface2 string) {
	defer stopWg.Done()
	stopWg.Add(9) // This function, 8x goroutines

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
	<-stop
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
