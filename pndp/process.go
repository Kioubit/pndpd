package pndp

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

// WaitForSignal Waits (blocking) for the program to be interrupted by the OS and then gracefully shuts down releasing all resources
func WaitForSignal() {
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	Shutdown()
}

// Shutdown Exits the program gracefully and releases all resources
//
//Do not use with WaitForSignal
func Shutdown() {
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

// SimpleRespond (Non blocking)
//
// iface - The interface to listen to and respond from
//
// filter - Optional (can be nil) list of CIDRs to whitelist. Must be IPV6!
// ParseFilter verifies ipv6
func SimpleRespond(iface string, filter []*net.IPNet) {
	go simpleRespond(iface, filter)
}

func simpleRespond(iface string, filter []*net.IPNet) {
	defer stopWg.Done()
	stopWg.Add(3) // This function, 2x goroutines
	requests := make(chan *ndpRequest, 100)
	defer close(requests)
	go respond(iface, requests, ndp_ADV, filter)
	go listen(iface, requests, ndp_SOL)
	<-stop
}

// Proxy NDP between interfaces iface1 and iface2
//
// Non blocking
func Proxy(iface1, iface2 string) {
	go proxy(iface1, iface2)
}

func proxy(iface1, iface2 string) {
	defer stopWg.Done()
	stopWg.Add(9) // This function, 8x goroutines

	req_iface1_sol_iface2 := make(chan *ndpRequest, 100)
	defer close(req_iface1_sol_iface2)
	go listen(iface1, req_iface1_sol_iface2, ndp_SOL)
	go respond(iface2, req_iface1_sol_iface2, ndp_SOL, nil)

	req_iface2_sol_iface1 := make(chan *ndpRequest, 100)
	defer close(req_iface2_sol_iface1)
	go listen(iface2, req_iface2_sol_iface1, ndp_SOL)
	go respond(iface1, req_iface2_sol_iface1, ndp_SOL, nil)

	req_iface1_adv_iface2 := make(chan *ndpRequest, 100)
	defer close(req_iface1_adv_iface2)
	go listen(iface1, req_iface1_adv_iface2, ndp_ADV)
	go respond(iface2, req_iface1_adv_iface2, ndp_ADV, nil)

	req_iface2_adv_iface1 := make(chan *ndpRequest, 100)
	defer close(req_iface2_adv_iface1)
	go listen(iface2, req_iface2_adv_iface1, ndp_ADV)
	go respond(iface1, req_iface2_adv_iface1, ndp_ADV, nil)
	<-stop
}

// ParseFilter Helper Function to Parse a string of CIDRs separated by a semicolon as a Whitelist for SimpleRespond
func ParseFilter(f string) []*net.IPNet {
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
