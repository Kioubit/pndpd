# PNDPD - NDP Responder + Proxy
## Features
- Efficiently process incoming packets using bpf (which runs in the kernel)
- Respond to all NDP solicitations on an interface
- Respond to NDP solicitations for whitelisted addresses on an interface
- Proxy NDP between interfaces with an optional whitelist for neighbor solicitations
- Optionally determine whitelist automatically based on the IPs assigned to the interfaces 
- Permissions required: root or CAP_NET_RAW

## Usage
```` 
pndpd config <path to file>
pndpd respond <interface> <optional whitelist of CIDRs separated by a semicolon>
pndpd proxy <interface1> <interface2> <optional whitelist of CIDRs separated by a semicolon applied to interface2>
````
More options and additional documentation in the example config file (pndpd.conf).

### Developing
It is easy to add functionality to PNDPD. For additions outside the core functionality you only need to keep the following methods in mind:
```` 
package main
import "pndpd/pndp"

pndp.ParseFilter(f string) []*net.IPNet

responderInstance := pndp.NewResponder(iface string, filter []*net.IPNet, autosenseInterface string)
responderInstance.Start()
responderInstance.Stop()

proxyInstance := pndp.NewProxy(iface1 string, iface2 string, filter []*net.IPNet, autosenseInterface string)
proxyInstance.Start()
proxyInstance.Stop()
````
Pull requests are welcome for any functionality you add.
