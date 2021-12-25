# PNDPD - NDP Responder + Proxy
## Features
- Efficiently process incoming packets using bpf (which runs in the kernel)
- Respond to all NDP solicitations on an interface
- Respond to NDP solicitations for whitelisted addresses on an interface
- Proxy NDP between interfaces with an optional whitelist
- Optionally determine whitelist automatically based on the IPs assigned to the interfaces 
- Permissions required: root or CAP_NET_RAW

## Installing & Updating

1) Download the latest release from the releases page and move the binary to the ``/urs/bin/`` 
2) For systemd users: Install the service
```` 
wget https://git.dn42.dev/Kioubit/Pndpd/src/branch/master/pndpd.service
mv pndpd.service /usr/lib/systemd/system/
systemctl enable pndpd.service
```` 
3) Download and install the config file
```` 
wget https://git.dn42.dev/Kioubit/Pndpd/src/branch/master/pndpd.conf
mkdir -p /etc/pndpd/
mv pndpd.conf /etc/pndpd/
````
4) Edit the config at ``/etc/pndpd/pndpd.conf`` and then start the service using ``service pndpd start``

## Manual Usage
```` 
pndpd config <path to file>
pndpd respond <interface> <optional whitelist of CIDRs separated by a semicolon>
pndpd proxy <interface1> <interface2> <optional whitelist of CIDRs separated by a semicolon applied to interface2>
````
More options and additional documentation in the example config file (pndpd.conf).

## Developing
### Adding Modules 
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
New functionality should be implemented as a module. You will find an example module under ``modules/example/``. 

Pull requests are welcome for any functionality you add.
