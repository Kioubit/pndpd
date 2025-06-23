#!/usr/bin/env sh
set -euxo

# Testing setup diagram
#
# +-------------------------------------------------------+
# |                  Provider Namespace (pndpd-p)         |
# |                                                       |
# |   +-----------------------------------------------+   |
# |   |                  Bridge (br0)                 |   |
# |   |  - fd00::/62                                  |   |
# |   |  - 2001:db8::/62                              |   |
# |   |  - MAC: 00:00:00:00:00:01                     |   |
# |   +-----------------------------------------------+   |
# |            |                                          |
# |            | veth-c0 (part of br0, 00:00:00:00:01:01) |
# |            |                                          |
# +------------|------------------------------------------+
#              |
#              | veth-p (00:00:00:00:02:02)
# +------------|------------------------------------------+
# |                  Client Namespace (pndpd-c)           |
# |                                                       |
# |   +-----------------------------------------------+   |
# |   |                  Bridge (br0)                 |   |
# |   |  - fd00:0:0:1::/64                            |   |
# |   |  - 2001:db8:0:1::/64                          |   |
# |   |  - MAC: 00:00:00:00:00:02                     |   |
# |   +-----------------------------------------------+   |
# |            |                                          |
# |            | veth-i0 (part of br0, 00:00:00:00:03:01) |
# |            |                                          |
# +------------|------------------------------------------+
#              |
#              | veth-c (00:00:00:00:03:02)
# +------------|------------------------------------------+
# |                  Inner Client Namespace (pndpd-i)     |
# |                                                       |
# |   - veth-c:                                           |
# |     - fd00:0:0:1::100/64                              |
# |     - 2001:db8:0:1::100/64                            |
# |   - Default route via fd00:0:0:1::                    |
# |                                                       |
# +-------------------------------------------------------+
#


if [ -z "${1:-}" ]; then
  echo "specify proxy or responder"
  exit
fi


case "$1" in
  "proxy")
    pndpd_command="ip netns exec pndpd-c bin/pndpd proxy veth-p br0 auto"
    ;;
  "responder")
    pndpd_command="ip netns exec pndpd-c bin/pndpd responder veth-p auto"
    ;;
  *)
    echo "Error: Invalid mode specified: '$1'.  Choose 'proxy' or 'responder'."
    exit 1
    ;;
esac


make

ip netns add pndpd-p # Provider
ip netns exec pndpd-p sysctl -w net.ipv6.conf.all.forwarding=1
ip netns add pndpd-c # Client
ip netns exec pndpd-c sysctl -w net.ipv6.conf.all.forwarding=1
ip netns add pndpd-i # Inner client
ip netns exec pndpd-i sysctl -w net.ipv6.conf.all.forwarding=1

ip -netns pndpd-p link set up lo
ip -netns pndpd-c link set up lo
ip -netns pndpd-i link set up lo

# Provider bridge
ip -netns pndpd-p link add br0 type bridge
ip -netns pndpd-p addr add fd00::/62 dev br0
ip -netns pndpd-p addr add 2001:db8::/62 dev br0
ip -netns pndpd-p link set dev br0 address 00:00:00:00:00:01 # Predictable link-local
ip -netns pndpd-p link set up dev br0

# Client 0 link to provider bridge
ip -netns pndpd-p link add veth-c0 type veth peer name veth-p
ip -netns pndpd-p link set dev veth-c0 address 00:00:00:00:01:01
ip -netns pndpd-p link set up veth-c0
ip -netns pndpd-p link set veth-c0 master br0

ip -netns pndpd-p link set veth-p netns pndpd-c
ip -netns pndpd-c addr add fd00:0:0:1::/128 dev veth-p
ip -netns pndpd-c addr add 2001:db8:0:1::/128 dev veth-p
ip -netns pndpd-c link set dev veth-p address 00:00:00:00:02:02
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
ip -netns pndpd-c link set dev veth-i0 address 00:00:00:00:03:01
ip -netns pndpd-c link set up veth-i0
ip -netns pndpd-c link set veth-i0 master br0

ip -netns pndpd-c link set veth-c netns pndpd-i
ip -netns pndpd-i addr add fd00:0:0:1::100/64 dev veth-c
ip -netns pndpd-i addr add 2001:db8:0:1::100/64 dev veth-c
ip -netns pndpd-i link set dev veth-c address 00:00:00:00:03:02
ip -netns pndpd-i link set up dev veth-c
ip -netns pndpd-i route add default via fd00:0:0:1:: dev veth-c

function teardown {
  echo "Performing teardown..."
  if [[ -n "$PID" ]]; then
    kill -n 2 "$PID" || echo "Error stopping program"
  fi

  if [[ -n "$PIDCAPTURE" ]]; then
      kill -n 2 "$PIDCAPTURE" || echo "Error stopping program"
  fi

  wait

  ip netns delete pndpd-p
  ip netns delete pndpd-c
  ip netns delete pndpd-i
}

trap teardown EXIT

sleep 2

ip netns exec pndpd-p tcpdump -i br0 -w test-result.pcap &
PIDCAPTURE=$!

$pndpd_command &
PID=$!
echo "program running with PID ${PID}"
sleep 2

# Perform tests
ip netns exec pndpd-p ping -c 5 -w 10 fd00:0:0:1::100
ip netns exec pndpd-p ping -c 5 -w 10 2001:db8:0:1::100

echo "Tests successful"

