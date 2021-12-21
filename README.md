# PNDPD - NDP Responder
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
