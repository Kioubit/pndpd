# PNDPD - NDP Responder + Proxy
## Features
- Efficiently process incoming packets using bpf (which runs in the kernel)
- Respond to all NDP solicitations on an interface
- Respond to NDP solicitations for whitelisted addresses on an interface
- Proxy NDP between interfaces
- Permissions required: root or CAP_NET_RAW

## Usage
```` 
pndpd readconfig <path to file>
pndpd respond <interface> <optional whitelist of CIDRs separated with a semicolon>
pndpd proxy <interface1> <interface2>
````

### Developing
It is easy to add functionality to PNDPD. For additions outside the core functionality you only need to keep the following methods in mind:
```` 
package main
import "pndpd/pndp"

pndp.SimpleRespond(iface string, filter []*net.IPNet)

pndp.Proxy(iface1, iface2 string)
````
Pull request are welcome for any functionality you add.