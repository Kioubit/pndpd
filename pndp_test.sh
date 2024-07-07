#!/usr/bin/env sh
set -euxo

make

ip netns add pndpd-p # Provider
ip netns exec pndpd-p sysctl -w net.ipv6.conf.all.forwarding=1
ip netns add pndpd-c # Client
ip netns exec pndpd-c sysctl -w net.ipv6.conf.all.forwarding=1
ip netns add pndpd-i # Inner client
ip netns exec pndpd-i sysctl -w net.ipv6.conf.all.forwarding=1

# Provider bridge
ip -netns pndpd-p link add br0 type bridge
ip -netns pndpd-p addr add fd00::/62 dev br0
ip -netns pndpd-p addr add 2001:db8::/62 dev br0
ip -netns pndpd-p link set dev br0 address 00:00:00:00:00:01 # Predictable link-local
ip -netns pndpd-p link set up dev br0

# Client 0 link to provider bridge
ip -netns pndpd-p link add veth-c0 type veth peer name veth-p
ip -netns pndpd-p link set up veth-c0
ip -netns pndpd-p link set veth-c0 master br0

ip -netns pndpd-p link set veth-p netns pndpd-c
ip -netns pndpd-c addr add fd00:0:0:1::/128 dev veth-p
ip -netns pndpd-c addr add 2001:db8:0:1::/128 dev veth-p
ip -netns pndpd-c link set up dev veth-p
ip -netns pndpd-c route add default via fe80::200:ff:fe00:1 dev veth-p

# Bridge on client 0
ip -netns pndpd-c link add br0 type bridge
ip -netns pndpd-c addr add fd00:0:0:1::/64 dev br0
ip -netns pndpd-c addr add 2001:db8:0:1::/64 dev br0
ip -netns pndpd-c link set dev br0 address 00:00:00:00:00:02
ip -netns pndpd-c link set up dev br0

# Inner client link to client 0 bridge
ip -netns pndpd-c link add veth-i0 type veth peer name veth-c
ip -netns pndpd-c link set up veth-i0
ip -netns pndpd-c link set veth-i0 master br0

ip -netns pndpd-c link set veth-c netns pndpd-i
ip -netns pndpd-i addr add fd00:0:0:1::100/64 dev veth-c
ip -netns pndpd-i addr add 2001:db8:0:1::100/64 dev veth-c
ip -netns pndpd-i link set up dev veth-c
ip -netns pndpd-i route add default via fd00:0:0:1:: dev veth-c

sleep 2
ip netns exec pndpd-c bin/pndpd proxy veth-p br0 auto &
PID=$!
sleep 2

# Perform tests
ip netns exec pndpd-p ping -c 5 -w 10 fd00:0:0:1::100
ip netns exec pndpd-p ping -c 5 -w 10 2001:db8:0:1::100

kill -n 2 "$PID"
wait
echo "Tests successful"

# Teardown
ip netns delete pndpd-p
ip netns delete pndpd-c
ip netns delete pndpd-i