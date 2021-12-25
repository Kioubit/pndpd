package pndp

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

var GlobalDebug = false

type ResponderObj struct {
	stopChan  chan struct{}
	stopWG    *sync.WaitGroup
	iface     string
	filter    []*net.IPNet
	autosense string
}
type ProxyObj struct {
	stopChan  chan struct{}
	stopWG    *sync.WaitGroup
	iface1    string
	iface2    string
	filter    []*net.IPNet
	autosense string
}

// NewResponder
//
// iface - The interface to listen to and respond from
//
// filter - Optional (can be nil) list of CIDRs to whitelist. Must be IPV6! ParseFilter verifies ipv6
//
// With the optional autosenseInterface argument, the whitelist is configured based on the addresses assigned to the interface specified. This works even if the IP addresses change frequently.
// Start() must be called on the object to actually start responding
func NewResponder(iface string, filter []*net.IPNet, autosenseInterface string) *ResponderObj {
	fmt.Println("WARNING: You should use a whitelist for the responder unless you really know what you are doing")
	var s sync.WaitGroup
	return &ResponderObj{
		stopChan:  make(chan struct{}),
		stopWG:    &s,
		iface:     iface,
		filter:    filter,
		autosense: autosenseInterface,
	}
}
func (obj *ResponderObj) Start() {
	go obj.start()
}
func (obj *ResponderObj) start() {
	obj.stopWG.Add(1)
	requests := make(chan *ndpRequest, 100)
	defer func() {
		close(requests)
		obj.stopWG.Done()
	}()
	go respond(obj.iface, requests, ndp_ADV, nil, obj.filter, obj.autosense, obj.stopWG, obj.stopChan)
	go listen(obj.iface, requests, ndp_SOL, obj.stopWG, obj.stopChan)
	fmt.Println("Started responder instance on interface", obj.iface)
	<-obj.stopChan
}

//Stop a running Responder instance
// Returns false on error
func (obj *ResponderObj) Stop() bool {
	close(obj.stopChan)
	fmt.Println("Shutting down responder instance..")
	if wgWaitTimout(obj.stopWG, 10*time.Second) {
		fmt.Println("Done")
		return true
	} else {
		fmt.Println("Error shutting down instance")
		return false
	}
}

// NewProxy Proxy NDP between interfaces iface1 and iface2 with an optional filter (whitelist)
//
// filter - Optional (can be nil) list of CIDRs to whitelist. Must be IPV6! ParseFilter verifies ipv6
//
// With the optional autosenseInterface argument, the whitelist is configured based on the addresses assigned to the interface specified. This works even if the IP addresses change frequently.
//
// Start() must be called on the object to actually start proxying
func NewProxy(iface1 string, iface2 string, filter []*net.IPNet, autosenseInterface string) *ProxyObj {
	var s sync.WaitGroup
	return &ProxyObj{
		stopChan:  make(chan struct{}),
		stopWG:    &s,
		iface1:    iface1,
		iface2:    iface2,
		filter:    filter,
		autosense: autosenseInterface,
	}
}

func (obj *ProxyObj) Start() {
	go obj.start()
}
func (obj *ProxyObj) start() {
	obj.stopWG.Add(1)
	defer func() {
		obj.stopWG.Done()
	}()

	out_iface1_sol_questions_iface2_adv := make(chan *ndpQuestion, 100)
	defer close(out_iface1_sol_questions_iface2_adv)
	out_iface2_sol_questions_iface1_adv := make(chan *ndpQuestion, 100)
	defer close(out_iface2_sol_questions_iface1_adv)

	req_iface1_sol_iface2 := make(chan *ndpRequest, 100)
	defer close(req_iface1_sol_iface2)
	go listen(obj.iface1, req_iface1_sol_iface2, ndp_SOL, obj.stopWG, obj.stopChan)
	go respond(obj.iface2, req_iface1_sol_iface2, ndp_SOL, out_iface2_sol_questions_iface1_adv, obj.filter, obj.autosense, obj.stopWG, obj.stopChan)

	req_iface2_sol_iface1 := make(chan *ndpRequest, 100)
	defer close(req_iface2_sol_iface1)
	go listen(obj.iface2, req_iface2_sol_iface1, ndp_SOL, obj.stopWG, obj.stopChan)
	go respond(obj.iface1, req_iface2_sol_iface1, ndp_SOL, out_iface1_sol_questions_iface2_adv, nil, "", obj.stopWG, obj.stopChan)

	req_iface1_adv_iface2 := make(chan *ndpRequest, 100)
	defer close(req_iface1_adv_iface2)
	go listen(obj.iface1, req_iface1_adv_iface2, ndp_ADV, obj.stopWG, obj.stopChan)
	go respond(obj.iface2, req_iface1_adv_iface2, ndp_ADV, out_iface1_sol_questions_iface2_adv, nil, "", obj.stopWG, obj.stopChan)

	req_iface2_adv_iface1 := make(chan *ndpRequest, 100)
	defer close(req_iface2_adv_iface1)
	go listen(obj.iface2, req_iface2_adv_iface1, ndp_ADV, obj.stopWG, obj.stopChan)
	go respond(obj.iface1, req_iface2_adv_iface1, ndp_ADV, out_iface2_sol_questions_iface1_adv, nil, "", obj.stopWG, obj.stopChan)

	fmt.Println("Started Proxy instance for interfaces:", obj.iface1, "and", obj.iface2)
	<-obj.stopChan
}

//Stop a running Proxy instance
// Returns false on error
func (obj *ProxyObj) Stop() bool {
	close(obj.stopChan)
	fmt.Println("Shutting down proxy instance..")
	if wgWaitTimout(obj.stopWG, 10*time.Second) {
		fmt.Println("Done")
		return true
	} else {
		fmt.Println("Error shutting down instance")
		return false
	}
}

// ParseFilter Helper Function to Parse a string of CIDRs separated by a semicolon as a Whitelist for SimpleRespond
func ParseFilter(f string) []*net.IPNet {
	if f == "" {
		return nil
	}
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
