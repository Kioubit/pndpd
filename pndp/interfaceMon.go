package pndp

import (
	"net"
	"sync"
)

// TODO incomplete implementation
var (
	interfaceMonSync    sync.Mutex
	interfaceMonRunning bool = false
	startCount          int  = 0
	s                   chan interface{}
	u                   chan *interfaceAddressUpdate
)

func startInterfaceMon() {
	interfaceMonSync.Lock()
	defer interfaceMonSync.Unlock()
	if !interfaceMonRunning {
		u := make(chan *interfaceAddressUpdate, 10)
		s := make(chan interface{})
		err := getInterfaceUpdates(u, s)
		if err != nil {
			panic(err.Error())
		}
	}
	startCount++
	go getUpdates()
}

func stopInterfaceMon() {
	interfaceMonSync.Lock()
	defer interfaceMonSync.Unlock()
	startCount--
	if interfaceMonRunning && startCount <= 0 {
		if s != nil {
			close(s)
		}
	}
}

func getUpdates() {
	for {
		update := <-u
		if update == nil {
			//channel closed
			return
		}
		if update.NetworkFamily != IPv6 {
			continue
		}
		iface, err := net.InterfaceByIndex(update.InterfaceIndex)
		if err != nil {
			continue
		}

		srcIP := selectSourceIP(iface)
		monMutex.Lock()
		for i := range monInterfaceList {
			if monInterfaceList[i].iface.Name == iface.Name {
				oldMonIface := monInterfaceList[i]
				oldMonIface.sourceIP = srcIP
				if oldMonIface.autosense {
					oldMonIface.networks = getInterfaceNetworkList(iface)
				}
				break
			}
		}
		monMutex.Unlock()
	}
}

type monInterface struct {
	addCount  int
	sourceIP  []byte //TODO ULA and GUA
	networks  []*net.IPNet
	iface     *net.Interface
	autosense bool
}

var (
	monInterfaceList = make([]*monInterface, 0)
	monMutex         sync.RWMutex
)

func addInterfaceToMon(iface string, autosense bool) {
	if iface == "" {
		return
	}
	monMutex.Lock()
	defer monMutex.Unlock()

	niface, err := net.InterfaceByName(iface)
	if err != nil {
		panic(err.Error())
	}

	for i := range monInterfaceList {
		if monInterfaceList[i].iface.Name == niface.Name {
			oldMonIface := monInterfaceList[i]
			if autosense {
				oldMonIface.autosense = true
			}
			oldMonIface.addCount++
			return
		}
	}
	newMonIface := &monInterface{
		addCount:  1,
		autosense: autosense,
		iface:     niface,
	}
	newMonIface.sourceIP = selectSourceIP(niface)
	newMonIface.networks = getInterfaceNetworkList(niface)

	monInterfaceList = append(monInterfaceList, newMonIface)
}

func removeInterfaceFromMon(iface string) {
	monMutex.Lock()
	defer monMutex.Unlock()
	niface, err := net.InterfaceByName(iface)
	if err != nil {
		panic(err.Error())
	}
	for i := range monInterfaceList {
		if monInterfaceList[i].iface.Name == niface.Name {
			oldMonIface := monInterfaceList[i]
			oldMonIface.addCount--
			if oldMonIface.addCount <= 0 {
				monInterfaceList[i] = monInterfaceList[len(s)-1]
				monInterfaceList = monInterfaceList[:len(s)-1]
			}
			return
		}
	}
}

func getInterfaceInfo(iface *net.Interface) *monInterface {
	ifaceName := iface.Name
	monMutex.RLock()
	for i := range monInterfaceList {
		if monInterfaceList[i].iface.Name == ifaceName {
			return monInterfaceList[i]
		}
	}
	defer monMutex.RUnlock()
	return nil
}
