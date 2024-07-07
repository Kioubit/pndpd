package pndp

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type ResponderObj struct {
	stopChan          chan struct{}
	stopWG            *sync.WaitGroup
	iface             string
	filter            []*net.IPNet
	autosense         string
	monitorInterfaces bool
}
type ProxyObj struct {
	stopChan          chan struct{}
	stopWG            *sync.WaitGroup
	iface1            string
	iface2            string
	filter            []*net.IPNet
	autosense         string
	monitorInterfaces bool
}

// NewResponder
//
// iface - The interface to listen to and respond from
//
// filter - Optional (can be nil) list of IPv6 addresses in CIDR notation to whitelist. Must be IPV6. The ParseFilter function verifies that.
//
// With the optional "autosenseInterface" argument, the whitelist is configured based on the addresses assigned to the interface specified.
// This works even if the IP addresses change frequently.
//
// Start() must be called on the object to actually start responding
func NewResponder(iface string, filter []*net.IPNet, autosenseInterface string, monitorInterfaces bool) *ResponderObj {
	if filter == nil && autosenseInterface == "" {
		fmt.Println("WARNING: You should use a whitelist for the responder unless you really know what you are doing")
	}
	checkIsValidNetworkInterfaceFatal(iface, autosenseInterface)

	var s sync.WaitGroup
	return &ResponderObj{
		stopChan:          make(chan struct{}),
		stopWG:            &s,
		iface:             iface,
		filter:            filter,
		autosense:         autosenseInterface,
		monitorInterfaces: monitorInterfaces,
	}
}
func (obj *ResponderObj) Start() {
	go obj.start()
}
func (obj *ResponderObj) start() {
	obj.stopWG.Add(1)

	startInterfaceMon()

	addInterfaceToMon(obj.iface, obj.monitorInterfaces)
	addInterfaceToMon(obj.autosense, true)

	requests := make(chan *ndpRequest, 100)
	defer func() {
		close(requests)
		obj.stopWG.Done()
	}()
	go respond(obj.iface, requests, ndp_ADV, nil, obj.filter, obj.autosense, obj.stopWG, obj.stopChan)
	go listen(obj.iface, requests, ndp_SOL, obj.stopWG, obj.stopChan)
	fmt.Printf("Started responder instance on interface %s", obj.iface)
	fmt.Println()
	<-obj.stopChan

	removeInterfaceFromMon(obj.iface)
	removeInterfaceFromMon(obj.autosense)
	stopInterfaceMon()
}

// Stop a running Responder instance
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
// filter - Optional (can be nil) list of IPv6 addresses in CIDR notation to whitelist. Must be IPV6. The ParseFilter function verifies that.
//
// With the optional "autosenseInterface" argument, the whitelist is configured based on the addresses assigned to the interface specified.
// This works even if the IP addresses change frequently.
//
// Start() must be called on the object to actually start proxying
func NewProxy(iface1 string, iface2 string, filter []*net.IPNet, autosenseInterface string, monitorInterfaces bool) *ProxyObj {

	checkIsValidNetworkInterfaceFatal(iface1, iface2, autosenseInterface)

	var s sync.WaitGroup
	return &ProxyObj{
		stopChan:          make(chan struct{}),
		stopWG:            &s,
		iface1:            iface1,
		iface2:            iface2,
		filter:            filter,
		autosense:         autosenseInterface,
		monitorInterfaces: monitorInterfaces,
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

	startInterfaceMon()
	addInterfaceToMon(obj.iface1, obj.monitorInterfaces)
	addInterfaceToMon(obj.iface2, obj.monitorInterfaces)
	addInterfaceToMon(obj.autosense, true)

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

	fmt.Printf("Started Proxy instance on interfaces %s and %s (if enabled, the whitelist is applied on %s)", obj.iface1, obj.iface2, obj.iface2)
	fmt.Println()
	<-obj.stopChan

	removeInterfaceFromMon(obj.iface1)
	removeInterfaceFromMon(obj.iface2)
	removeInterfaceFromMon(obj.autosense)
	stopInterfaceMon()
}

// Stop a running Proxy instance
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

// ParseFilter Helper Function to Parse a string of CIDRs separated by a semicolon as a Whitelist
func ParseFilter(f string) []*net.IPNet {
	if f == "" {
		return nil
	}
	s := strings.Split(f, ";")
	result := make([]*net.IPNet, len(s))
	for i, n := range s {
		_, cidr, err := net.ParseCIDR(n)
		if err != nil {
			showFatalError("filter:", err.Error())
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

func isValidNetworkInterface(iface string) bool {
	if iface == "" {
		return true
	}
	if _, err := net.InterfaceByName(iface); err != nil {
		return false
	}
	return true
}

func checkIsValidNetworkInterfaceFatal(iface ...string) {
	for i := range iface {
		if !isValidNetworkInterface(iface[i]) {
			showFatalError(fmt.Sprintf("No such network interface \"%s\"", iface[i]))
		}
	}
}

func showFatalError(error ...string) {
	fmt.Printf("Error: ")
	for _, err := range error {
		fmt.Printf(err + " ")
	}
	fmt.Println()
	fmt.Println("Exiting due to error")
	os.Exit(1)
}
