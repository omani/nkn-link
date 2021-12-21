//go:build !windows

package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/lorenzosaino/go-sysctl"
	"github.com/omani/nkn-link/config"
	"github.com/omani/nkn-link/tun"
	"github.com/songgao/packets/ethernet"
	"github.com/vishvananda/netlink"
)

var opts struct {
	// Client bool `short:"c" long:"client" description:"Client mode"`
	// Server bool `short:"s" long:"server" description:"Server mode"`

	ConfigFile string `short:"f" long:"config-file" default:"config.yaml" description:"Config file path"`

	Debug   bool `long:"debug" description:"Enable debug mode"`
	Version bool `long:"version" description:"Print version"`
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("Panic: %+v", r)
		}
	}()

	_, err := flags.Parse(&opts)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		log.Fatal(err)
	}

	if opts.Version {
		fmt.Println(opts.Version)
		os.Exit(0)
	}

	conf, err := config.NewConfig(opts.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	seed, err := hex.DecodeString(conf.NKNAccountSeed)
	if err != nil {
		log.Fatal(err)
	}

	// open NKN account with given seed
	account, err := conf.NewAccount(seed)
	if err != nil {
		log.Fatal(err)
	}
	conf.Set("nkn_account_seed", hex.EncodeToString(account.Seed()))

	// create new NKN multiclient
	client, err := conf.NewMultiClient(account, config.IDENTIFIER, 1, true)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	tun_device, err := tun.CreateTUN(conf.TunDeviceName, config.DefaultMTU)
	if err != nil {
		log.Fatal(err)
	}
	defer tun_device.Close()

	tun_device_name, err := tun_device.Name()
	if err != nil {
		log.Fatal(err)
	}

	tun_link, err := netlink.LinkByName(tun_device_name)
	if err != nil {
		log.Fatal(err)
	}

	// set IP address of new TUN device
	addr, _ := netlink.ParseAddr(conf.TunDeviceIPAddress)
	netlink.AddrAdd(tun_link, addr)
	netlink.LinkSetUp(tun_link)
	defer netlink.LinkDel(tun_link)

	// don't care about the previous setting of ip_forward on this machine. just set if true.
	if conf.EnableIPForwarding {
		sysctl.Set("net.ipv4.ip_forward", "1")
	}

	var routes []netlink.Route

	// if `gateway` cli argument is set, change routing table accordingly.
	if conf.DefaultRouteEnable {
		gateway := conf.DefaultRouteGatewayAddress
		if len(gateway) > 0 {
			routelist, err := netlink.RouteList(nil, 4)
			if err != nil {
				log.Fatal(err)
			}
			// fetch default route (in linux it is always the most top (first) route in the route table)
			netlink.RouteDel(&routelist[0])
			// lower its metric so our new default route is higher
			routelist[0].Priority = 100
			netlink.RouteAdd(&routelist[0])

			// now add a new default route with our remote peer as the gateway
			routes = append(routes, netlink.Route{
				LinkIndex: tun_link.Attrs().Index,
				Dst: &net.IPNet{
					IP:   net.IPv4(0, 0, 0, 0),
					Mask: net.CIDRMask(0, 32),
				},
				Src:      net.ParseIP(conf.TunDeviceIPAddress),
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
			n, err := tun_device.Read([]byte(tx_frame), 0)
			if err != nil {
				log.Fatal(err)
			}
			tx_frame = tx_frame[:n]
			if opts.Debug {
				fmt.Println("----------------SENDING----------------")
				log.Printf("Dst: %s\n", tx_frame.Destination())
				log.Printf("Src: %s\n", tx_frame.Source())
				log.Printf("Ethertype: % x\n", tx_frame.Ethertype())
				log.Printf("Payload: % x\n", tx_frame.Payload())
				fmt.Printf("----------------------------------------\n\n")
			}

			_, err = client.Send(
				conf.GetNKNRemotePeer(),
				[]byte(tx_frame),
				nil,
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
		if opts.Debug {
			fmt.Println("----------------RECEIVED----------------")
			log.Printf("Dst: %s\n", rx_frame.Destination())
			log.Printf("Src: %s\n", rx_frame.Source())
			log.Printf("Ethertype: % x\n", rx_frame.Ethertype())
			log.Printf("Payload: % x\n", rx_frame.Payload())
			fmt.Printf("-----------------------------------------\n\n")
		}

		_, err := tun_device.Write([]byte(rx_frame), 0)
		if err != nil {
			log.Fatal(err)
		}
	}
}
