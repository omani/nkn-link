/*
	Author:			HAH! Sun (github.com/omani)
	Date:			2021-12-12 04:05
	Title:			NKN-Link
	Description:	A Point-to-Point (Layer 3) tunnel between NKN Peers.
					Create a layer 3 tunnel between two NKN peers.
	Note:			Written down in one go and thus a single file.
*/

package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	sysctl "github.com/lorenzosaino/go-sysctl"

	"github.com/nknorg/nkn-sdk-go"
	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
	"github.com/spf13/viper"
	"github.com/vishvananda/netlink"
)

const SEEDRPCSERVERADDR = "http://178.128.136.86:30003"
const IDENTIFIER = "nkn-link"

func main() {
	clientconfig := &nkn.ClientConfig{
		SeedRPCServerAddr:       nkn.NewStringArray(SEEDRPCSERVERADDR),
		RPCTimeout:              5000,
		ConnectRetries:          1,
		MsgCacheExpiration:      300000,
		MsgCacheCleanupInterval: 60000,
		WsHandshakeTimeout:      2500,
		WsWriteTimeout:          5000,
	}

	configpath := flag.String("config", ".", "Path to config file. Default is $CWD.")
	debug := flag.Bool("debug", false, "Enable debug output.")
	init := flag.Bool("init", false, "Initialize config.yaml.")
	flag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(*configpath)

	// init config.yaml with preconfigured settings:
	// - save seed of newly created NKN account
	// - print address of the NKN client
	if *init {
		fmt.Println(` _ _  _ __ _ _       _    _       _   
| \ || / /| \ | ___ | |  <_>._ _ | |__
|   ||  \ |   ||___|| |_ | || ' || / /
|_\_||_\_\|_\_|     |___||_||_|_||_\_\`)
		fmt.Printf("\n\nWelcome to NKN-Link. A Point-to-Point (Layer 3) tunnel between NKN Peers.\n\n\n")
		account, err := nkn.NewAccount(nil)
		if err != nil {
			log.Fatal(err)
		}

		client, err := nkn.NewMultiClient(account, IDENTIFIER, 1, true, clientconfig)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("NKN peer address: %s\n\n", client.Address())

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("NKN address of remote peer: ")
		remote_peer_addr, _ := reader.ReadString('\n')
		viper.Set("nkn_remote_peer", strings.TrimSuffix(remote_peer_addr, "\n"))

		fmt.Print("Choose IP address for TUN device in CIDR format (eg. 10.0.0.1/24): ")
		tun_device_ip_address, _ := reader.ReadString('\n')
		viper.Set("tun_device_ip_address", strings.TrimSuffix(tun_device_ip_address, "\n"))
		viper.Set("nkn_seedrpcserver_address", SEEDRPCSERVERADDR)
		viper.Set("nkn_account_seed", hex.EncodeToString(account.Seed()))
		viper.Set("tun_device_name", IDENTIFIER)
		viper.Set("default_route_enable", false)
		viper.Set("default_route_gateway_address", "")
		viper.Set("enable_ip_forwarding", false)

		err = viper.WriteConfigAs("config.yaml")
		if err != nil {
			log.Fatalf("Could not write to config.yaml: %v", err)
		}

		fmt.Println("To enable IP forwarding on this peer, set `enable_ip_forwarding` to `true`.")
		fmt.Println("Successfully written to config.yaml.")
		fmt.Println("All set. Run `sudo ./nkn-vpn` to start.")
		client.Close()
		return
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("%v.\nRun 'sudo ./nkn-vpn -init' to initialize a config file.\n", err)
	}

	// user preferred seed rpc server address
	if len(viper.GetString("nkn_seedrpcserver_address")) > 0 {
		clientconfig.SeedRPCServerAddr = nkn.NewStringArray(viper.GetString("nkn_seedrpcserver_address"))
	}

	// if `nkn_remote_peer` is missing in config file.
	if len(viper.GetString("nkn_remote_peer")) == 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("NKN address of remote peer: ")
		remote_peer_addr, _ := reader.ReadString('\n')
		viper.Set("nkn_remote_peer", strings.TrimSuffix(remote_peer_addr, "\n"))
	}

	seed, err := hex.DecodeString(viper.GetString("nkn_account_seed"))
	if err != nil {
		log.Fatal(err)
	}

	// open NKN account with given seed
	account, err := nkn.NewAccount(seed)
	if err != nil {
		log.Fatal(err)
	}
	viper.Set("nkn_account_seed", hex.EncodeToString(account.Seed()))
	viper.WriteConfig()

	// create new NKN multiclient
	client, err := nkn.NewMultiClient(account, IDENTIFIER, 1, true, clientconfig)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// create new TUN device
	config := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			MultiQueue: true,
			Name:       viper.GetString("tun_device_name"),
		},
	}
	tun_device, err := water.New(config)
	if err != nil {
		log.Fatal(err)
	}
	defer tun_device.Close()

	// set IP address of new TUN device
	tun_link, err := netlink.LinkByName(tun_device.Name())
	if err != nil {
		log.Fatal(err)
	}
	addr, _ := netlink.ParseAddr(viper.GetString("tun_device_ip_address"))
	netlink.AddrAdd(tun_link, addr)
	netlink.LinkSetUp(tun_link)
	defer netlink.LinkDel(tun_link)

	// don't care about the previous setting of ip_forward on this machine. just set if true.
	if viper.GetBool("enable_ip_forwarding") {
		sysctl.Set("net.ipv4.ip_forward", "1")
	}

	var routes []netlink.Route

	// if `gateway` cli argument is set, change routing table accordingly.
	if viper.GetBool("default_route_enable") {
		gateway := viper.GetString("default_route_gateway_address")
		if len(gateway) > 0 {
			routelist, err := netlink.RouteList(nil, 4)
			if err != nil {
				log.Fatal(err)
			}
			// fetch default route (in linux it is always the most top (first) route in the route table)
			netlink.RouteDel(&routelist[0])
			// lower its metric so our new default route is higher
			routelist[0].Priority = 100
			netlink.RouteReplace(&routelist[0])

			// now add a new default route with our remote peer as the gateway
			routes = append(routes, netlink.Route{
				LinkIndex: tun_link.Attrs().Index,
				Dst: &net.IPNet{
					IP:   net.IPv4(0, 0, 0, 0),
					Mask: net.CIDRMask(0, 32),
				},
				Src:      net.ParseIP(viper.GetString("tun_device_ip_address")),
				Priority: 0,
				Table:    routelist[0].Table,
				Gw:       net.ParseIP(gateway),
			})

			// iterate over node updates
			// keep already established rpc node connection available via current gateway
			go func() {
				// add all populated routes
				for _, route := range routes {
					if err := netlink.RouteAdd(&route); err != nil {
						log.Fatal(err)
					}
				}

				for {
					conn := <-client.OnConnect.C
					rpc_node := strings.Split(conn.Addr, ":")[0]
					route := netlink.Route{
						LinkIndex: routelist[0].LinkIndex,
						Dst: &net.IPNet{
							IP:   net.ParseIP(rpc_node),
							Mask: net.CIDRMask(32, 32),
						},
						Src:      routelist[0].Src,
						Priority: routelist[0].Priority,
						Table:    routelist[0].Table,
						Gw:       routelist[0].Gw,
					}
					if err := netlink.RouteAdd(&route); err != nil {
						log.Fatal(err)
					}
					routes = append(routes, route)
				}
			}()
		} else {
			log.Println("`default_route_gateway_address` not set in config.yaml. Skipping.")
		}
	}

	// handle SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("Cleanup.")
		for _, route := range routes {
			netlink.RouteDel(&route)
		}
		fmt.Println("Exiting.")
		os.Exit(1)
	}()

	var tx_frame ethernet.Frame
	var rx_frame ethernet.Frame

	// sender (tx)
	go func() {
		for {
			tx_frame.Resize(1500)
			n, err := tun_device.Read([]byte(tx_frame))
			if err != nil {
				log.Fatal(err)
			}
			tx_frame = tx_frame[:n]
			if *debug {
				fmt.Println("----------------SENDING----------------")
				log.Printf("Dst: %s\n", tx_frame.Destination())
				log.Printf("Src: %s\n", tx_frame.Source())
				log.Printf("Ethertype: % x\n", tx_frame.Ethertype())
				log.Printf("Payload: % x\n", tx_frame.Payload())
				packet := gopacket.NewPacket(tx_frame.Payload(), layers.LayerTypeEthernet, gopacket.Lazy)
				spew.Dump(packet)
				fmt.Printf("----------------------------------------\n\n")
			}

			_, err = client.Send(
				nkn.NewStringArrayFromString(viper.GetString("nkn_remote_peer")),
				[]byte(tx_frame),
				&nkn.MessageConfig{NoReply: false},
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	// receiver (rx)
	for {
		msg := <-client.OnMessage.C
		rx_frame = msg.Data
		if *debug {
			fmt.Println("----------------RECEIVED----------------")
			log.Printf("Dst: %s\n", rx_frame.Destination())
			log.Printf("Src: %s\n", rx_frame.Source())
			log.Printf("Ethertype: % x\n", rx_frame.Ethertype())
			log.Printf("Payload: % x\n", rx_frame.Payload())
			packet := gopacket.NewPacket(rx_frame.Payload(), layers.LayerTypeEthernet, gopacket.Lazy)
			spew.Dump(packet)
			fmt.Printf("-----------------------------------------\n\n")
		}

		_, err := tun_device.Write([]byte(rx_frame))
		if err != nil {
			log.Fatal(err)
		}
	}
}
