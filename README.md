# PNDPD - NDP Proxy / Responder (IPv6)
## Features
- **Efficiently** process incoming packets using bpf (which runs in the kernel)
- **Proxy** NDP between interfaces with an optional whitelist
- Optionally determine whitelist **automatically** based on the IPs assigned to the interfaces
- **Respond** to NDP solicitations for all or only whitelisted addresses on an interface
- Permissions required: root or **CAP_NET_RAW**
- Easily expandable with modules

## Installing & Updating

1) Download the latest release from the [releases page](https://github.com/Kioubit/pndpd/releases) and move the binary to the ``/usr/local/bin/`` directory under the filename ``pndpd``.
2) Allow executing the file by running ``chmod +x /usr/local/bin/pndpd``
3) Install the systemd service unit file:
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
pndpd proxy <external interface> <internal interface> <[optional] 'auto' to determine filters from the internal interface or whitelist of CIDRs separated by a semicolon>
pndpd responder <external interface> <[optional] 'auto' to determine filters from the external interface or whitelist of CIDRs separated by a semicolon>
pndpd config <path to file>
````
**Example:** ``pndpd proxy eth0 tun0 auto``

Find more options and additional documentation in the example config file (``pndpd.conf``).

## Example Scenario
### Proxying NDP requests for a /64 IPv6 subnet on a VPS to a VPN tunnel 

#### 1) Inspecting the initial IP configuration
````
root@vultr:~# ip -6 addr show dev enp1s0
2: enp1s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
    inet6 2001:11ff:7400:82f2:5400:4ff:fe53:26cf/64 scope global dynamic mngtmpaddr 
       valid_lft 2591753sec preferred_lft 604553sec
    inet6 fe80::5400:4ff:fe53:26cf/64 scope link 
       valid_lft forever preferred_lft forever
```` 
As we can see from the output, a `/64` subnet of public IPv6 addresses has been assigned to our VPS on our WAN interface `enp1s0`:
`2001:11ff:7400:82f2:5400:4ff:fe53:26cf/64`.

#### 2) Routing the subnet to the VPN interface
To route this subnet to our VPN interface `tun0` we need to assign one ip address to the VPS and the rest to the VPN interface.  
To do that we edit the `/etc/network/interface` file (for systems that use ifupdown2):

##### Initial contents:
````
allow-hotplug enp1s0

iface enp1s0 inet static 
    #.... IPv4 config here ...

iface enp1s0 inet6 static
    address 2001:11ff:7400:82f2:5400:4ff:fe53:26cf/64
    gateway fe80::fc00:4ff:fe53:26cf
````
##### After editing:
````
allow-hotplug enp1s0

iface enp1s0 inet static 
    #.... IPv4 config here ...

iface enp1s0 inet6 static
    address 2001:11ff:7400:82f2::1/128
    gateway fe80::fc00:4ff:fe53:26cf
````
On the VPN interface we can now assign the rest of the addresses:

`ip addr add 2001:11ff:7400:82f2::1/64 dev tun0`

#### 3) Running PNDPD
To proxy NDP requests from the outside interface to the VPN interface we run pndp like this:
````
sudo pndpd proxy enp1s0 tun0 auto
````
Note: sudo is not required if you are using the capability as described in the systemd unit file.
Optionally confirm that the setup works via ping and tcpdump.  

## Building PNDPD
For building, the version of go needs to be installed that is specified in the `go.mod` file. A makefile is available. Optionally adjust the ``MODULES`` variable to include or exclude modules from the "modules" directory.
````
make clean; make release
```` 
Find the binaries in the ``bin/`` directory
