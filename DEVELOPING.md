## Developing
### Adding Modules
New functionality should be implemented as a module where possible. You will find an example module under ``modules/example/``.
For additions outside the core functionality you only need to keep the following methods in mind:
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
 
