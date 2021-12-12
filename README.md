<div style="text-align: justify">

NKN-Link
===

<p align="center"><img src="images/logo.png" alt="NKN-Link"/></p>

A Point-to-Point (Layer 3) tunnel between NKN Peers.

# NKN-Link

[![GoDoc](https://godoc.org/github.com/omani/nkn-link?status.svg)](https://godoc.org/github.com/omani/nkn-link)

## Table of Contents
- [Description](#Description)
- [Install](#Install)
- [Setup](#Setup)
- [Run](#Run)
- [Performance](#Performance)
- [Contribution](#Contribution)
- [Donations](#Donations)
- [Author](#Author)
- [License](#License)


## Description
NKN-Link provides a layer 3, point-to-point tunnel between two peers on the NKN network. This enables both peers to 
talk over IP and provides the ability to do whatever one can do with IP connectivity. NKN-Link can be seen as a simple 
alternative to OpenVPN or Wireguard. The code is very simple and began as a fun pet project. I use NKN-Link as my VPN 
solution at home and for my laptop, by using a VPS as my remote NKN peer.

The code makes use of the NKN MultiClient, but without sessions (no transmission and congestion control).
That means, that this tunnel is similar to a UDP connection.

A TUN device is created and each peer can talk via IP packets over the internet on an encrypted communication
channel provided by the NKN network, `nkn-link` does not come with its own encryption logic. The encryption is 
completely handled by NKN itself.

The code was written in one go and comes in a single file. No OOP-style logic.

This tool requires root privileges (`sudo`) to be able to create the TUN device.

Tested only on GNU/Linux. More builds and tests will follow.

### Use Cases
- Connect a VPS to the home network
- Securely route all traffic from home over a VPS in a datacenter
- Quickly spin up a tunnel to reach another peer over SSH
- Connect with a friend over NKN to exchange files
- Upload files to a remote storage
- Make a friend in an oppressed country reach the web by providing an "exit node"
- Connect multiple NKN peers together to form a private network
- Do whatever one can do with IP connectivity

---

## Install
```
go install github.com/omani/nkn-link
```

### Build from source
```
go get github.com/omani/nkn-link
make build
```

## Setup
This example will:
- use 2 NKN peers (A and B) on the internet
- show how to create an IP link between both
- show how to enable IP forwarding on peer A
- show how to use peer A as a default route for peer B to route all of its traffic

### Initialize configuration
If `nkn-link` cannot find a `config.yaml` file it will start in interactive mode to initialize one:
```
peerA $ sudo $GOPATH/bin/nkn-link
 _ _  _ __ _ _       _    _       _   
| \ || / /| \ | ___ | |  <_>._ _ | |__
|   ||  \ |   ||___|| |_ | || ' || / /
|_\_||_\_\|_\_|     |___||_||_|_||_\_\


Welcome to NKN-Link. A Point-to-Point (Layer 3) tunnel between NKN Peers.

NKN peer address: nkn-link.ab8d73d580e5fcdfaad25ca442dd33066a22b9a7d28809acc5fb3f6ff39666d0

NKN address of remote peer: 
```
```
peerB $ sudo $GOPATH/bin/nkn-link
 _ _  _ __ _ _       _    _       _   
| \ || / /| \ | ___ | |  <_>._ _ | |__
|   ||  \ |   ||___|| |_ | || ' || / /
|_\_||_\_\|_\_|     |___||_||_|_||_\_\


Welcome to NKN-Link. A Point-to-Point (Layer 3) tunnel between NKN Peers.

NKN peer address: nkn-link.24cb341473d3c1753611beda13625e2fbfec247141e916753b6e75ed204c42e3

NKN address of remote peer: 
```

Copy paste each address into the other peer's input `NKN address of remote peer:` and choose an IP address for your
peer.

#### Peer A:
```
Welcome to NKN-Link. A Point-to-Point (Layer 3) tunnel between NKN Peers.

NKN peer address: nkn-link.ab8d73d580e5fcdfaad25ca442dd33066a22b9a7d28809acc5fb3f6ff39666d0

NKN address of remote peer: nkn-link.24cb341473d3c1753611beda13625e2fbfec247141e916753b6e75ed204c42e3
IP address for TUN device in CIDR format (eg. 10.0.0.1/24): 10.0.0.1/24
To enable IP forwarding on this peer, set `enable_ip_forwarding` to `true`.
Successfully written to config.yaml.
All set. Run `sudo ./nkn-link` to start.
```
#### Peer B:
```
Welcome to NKN-Link. A Point-to-Point (Layer 3) tunnel between NKN Peers.

NKN peer address: nkn-link.24cb341473d3c1753611beda13625e2fbfec247141e916753b6e75ed204c42e3

NKN address of remote peer: nkn-link.ab8d73d580e5fcdfaad25ca442dd33066a22b9a7d28809acc5fb3f6ff39666d0
IP address for TUN device in CIDR format (eg. 10.0.0.1/24): 10.0.0.2/24
To enable IP forwarding on this peer, set `enable_ip_forwarding` to `true`.
Successfully written to config.yaml.
All set. Run `sudo ./nkn-link` to start.
```

Both peers have been initialized with a config file named `config.yaml` in the current directory.

Example `config.yaml` of peer A:
```
default_route_enable: false
default_route_gateway_address: ""
enable_ip_forwarding: false
nkn_account_seed: bec785fbd97f5a1287f59ce21ab10d485b3f76802f126d0e2aea82fc5f0e4170
nkn_remote_peer: nkn-link.ab8d73d580e5fcdfaad25ca442dd33066a22b9a7d28809acc5fb3f6ff39666d0
nkn_seedrpcserver_address: http://178.128.136.86:30003
tun_device_ip_address: 10.0.0.1/24
tun_device_name: nkn-link
```

**Note**: If a custom `nkn_seedrpcserver_address` is desired, it should be in IP format rather than DNS.

## Run
Ready to start `nkn-link` as sudo:
```
sudo $GOPATH/bin/nkn-link
```

Peer A and Peer B should be able to ping each other now:
```
peerA $: ping -c3 10.0.0.2
PING 10.0.0.2 (10.0.0.2) 56(84) bytes of data.
64 bytes from 10.0.0.2: icmp_seq=1 ttl=64 time=345 ms
64 bytes from 10.0.0.2: icmp_seq=2 ttl=64 time=346 ms
64 bytes from 10.0.0.2: icmp_seq=3 ttl=64 time=359 ms

--- 10.0.0.2 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2003ms
rtt min/avg/max/mdev = 344.750/350.031/358.898/6.307 ms
```
**Note**:
Traffic is encrypted per default in NKN. The roundtrip time can vary depending on the actual route it takes within NKN.

### Enable IP Forwarding
To enable IP Forwarding, set `enable_ip_forwarding` to `true`.

### Enable default route
Given that peer A has IP Forwarding enabled, peer B can now route all its traffic through peer A.

Set default gateway in `config.yaml` to IP address of peer A:
```
default_route_gateway_address: 10.0.0.1
```

Enable default route in `config.yaml`:
```
default_route_enable: true
```

Start `nkn-link`:
```
sudo $GOPATH/bin/nkn-link
```

**Note**:
Setting `default_route_enable` to `true` will alter the route table of the peer. The default route will be kept, but 
its metric will be increased. The active connection to the NKN seed node and any incoming RPC node update will be added 
to the old default route so that connectivity is ensured. After that, `default_route_gateway_address` will be set as 
the new gateway for the default route. The new route will have a lower metric, so that traffic will flow over it.

Example of route table of peer B after enabling `default_route_enable`:
```
peerB $ ip r
default via 10.0.0.1 dev nkn-link
default via 192.168.1.1 dev ens3 metric 100 
10.0.0.0/24 dev nkn-link proto kernel scope link src 10.0.0.2 
```
The previous default route with gateway `192.168.1.1` has its metric increased to 100. A new default route has been 
added with gateway `default_route_gateway_address` over the newly created TUN device `nkn-link`.

A simple `curl ifconfig.me` on peer B will now output the public IP address of the remote peer A.

## Performance

### Speed comparison
Scenario: 2 peers with a 100Mbps uplink, each. One uses the other's link as a hop. Download a 100MB file.

`curl` with regular uplink (internet connection):
```
$ curl -o bigfile https://speed.hetzner.de/100MB.bin
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  100M  100  100M    0     0  9.3M      0  0:00:08  0:00:08 --:--:-- 9.3M
```
Average download speed: 9.3MB/s. 8 seconds.

`curl` with routing traffic over remote peer using `nkn-link`:
```
$ curl --interface nkn-link -o bigfile https://speed.hetzner.de/100MB.bin
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  100M  100  100M    0     0   615k      0  0:02:46  0:02:46 --:--:-- 1292k
```
Average download speed: 615kB/s. 2:46 minutes.

#### Result
The connection speed when using the tunnel drops by around 15x (equal to a 6Mbps connection).

Due to the nature of NKN and how it works, the connection speed will vary.

---

## Contribution
* You can fork this, extend it and contribute back.
* You can contribute with pull requests.

## Donations
You can make me happy by donating NKN to the following address:
```
NKNVmZQZcDrgdMJKdgRfz2gn5ZdTAyro5uHm
```

## Author
HAH! Sun (omani)

## LICENSE
MIT License

</div>