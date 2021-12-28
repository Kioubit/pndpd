# PNDPD - NDP Responder + Proxy
## Features
- **Efficiently** process incoming packets using bpf (which runs in the kernel)
- Respond to all NDP solicitations on an interface
- **Respond** to NDP solicitations for whitelisted addresses on an interface
- **Proxy** NDP between interfaces with an optional whitelist
- Optionally determine whitelist **automatically** based on the IPs assigned to the interfaces 
- Permissions required: root or CAP_NET_RAW
- Easily expandable with modules

## Installing & Updating

1) Download the latest release from the releases page and move the binary to the ``/usr/local/bin/`` directory under the filename ``pndpd``.
2) Allow executing the file by running ``chmod +x /usr/local/bin/pndpd``
3) **For systemd users:** Install the service unit file
```` 
wget https://raw.githubusercontent.com/Kioubit/pndpd/master/pndpd.service -P /etc/systemd/system/
systemctl enable pndpd.service
```` 
4) Download and install the config file
```` 
mkdir -p /etc/pndpd
wget https://raw.githubusercontent.com/Kioubit/pndpd/master/pndpd.conf -P /etc/pndpd/
````
5) Edit the config at ``/etc/pndpd/pndpd.conf`` and then start the service using ``service pndpd start``

## Manual Usage
```` 
pndpd config <path to file>
pndpd responder <interface> <optional whitelist of CIDRs separated by a semicolon>
pndpd proxy <interface1> <interface2> <optional whitelist of CIDRs separated by a semicolon applied to interface2>
````
More options and additional documentation in the example config file (``pndpd.conf ``).

## Developing

### Building
For building, the version of go needs to be installed that is specified in the go.mod file. A makefile is available. Optionally adjust the modules variable to include/exclude modules from the modules directory.
````
make build
make release
```` 
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
