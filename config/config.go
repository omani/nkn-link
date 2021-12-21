package config

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/nknorg/nkn-sdk-go"
	"github.com/spf13/viper"
)

const SEEDRPCSERVERADDR = "http://178.128.136.86:30003"
const IDENTIFIER = "nkn-link"
const DefaultMTU = 1420

type Config struct {
	path            string
	NKNClientConfig *nkn.ClientConfig

	DefaultRouteEnable         bool             `yaml:"default_route_enable"`
	DefaultRouteGatewayAddress string           `yaml:"default_route_gateway_address"`
	EnableIPForwarding         bool             `yaml:"enable_ip_forwarding"`
	NKNAccountSeed             string           `yaml:"nkn_account_seed"`
	NKNRemotePeer              string           `yaml:"nkn_remote_peer"`
	NKNSeedRPCServerAddress    *nkn.StringArray `yaml:"nkn_seedrpcserver_address"`
	TunDeviceIPAddress         string           `yaml:"tun_device_ip_address"`
	TunDeviceName              string           `yaml:"tun_device_name"`
}

func NewConfig(path string) (*Config, error) {
	nkn_clientconfig := &nkn.ClientConfig{
		SeedRPCServerAddr:       nkn.NewStringArray(SEEDRPCSERVERADDR),
		RPCTimeout:              5000,
		ConnectRetries:          1,
		MsgCacheExpiration:      300000,
		MsgCacheCleanupInterval: 60000,
		WsHandshakeTimeout:      2500,
		WsWriteTimeout:          5000,
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(path)

		err := viper.ReadInConfig()
		if err != nil {
			fmt.Println(` _ _  _ __ _ _       _    _       _   
| \ || / /| \ | ___ | |  <_>._ _ | |__
|   ||  \ |   ||___|| |_ | || ' || / /
|_\_||_\_\|_\_|     |___||_||_|_||_\_\`)
			fmt.Printf("\n\nWelcome to NKN-Link. A Point-to-Point (Layer 3) tunnel between NKN Peers.\n\n")
			account, err := nkn.NewAccount(nil)
			if err != nil {
				log.Fatal(err)
			}

			client, err := nkn.NewMultiClient(account, IDENTIFIER, 1, true, nkn_clientconfig)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("NKN peer address: %s\n\n", client.Address())
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("NKN address of remote peer: ")
			remote_peer_addr, _ := reader.ReadString('\n')
			remote_peer_addr = strings.TrimSuffix(remote_peer_addr, "\n")
			viper.Set("nkn_remote_peer", remote_peer_addr)

			fmt.Print("IP address for TUN device in CIDR format (eg. 10.0.0.1/24): ")
			tun_device_ip_address, _ := reader.ReadString('\n')
			tun_device_ip_address = strings.TrimSuffix(tun_device_ip_address, "\n")
			if len(tun_device_ip_address) == 0 {
				tun_device_ip_address = "10.0.0.1/24"
			}

			viper.Set("tun_device_ip_address", tun_device_ip_address)
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

			fmt.Println("Successfully written to config.yaml.")
			fmt.Println("All set. Run `sudo ./nkn-link` to start.")
			client.Close()
		}

		c := &Config{
			path:            path,
			NKNClientConfig: nkn_clientconfig,
		}

		err = json.Unmarshal(b, c)
		if err != nil {
			return nil, err
		}

		return c, nil
	}

	c := &Config{
		path: path,
	}

	err = json.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	// user preferred seed rpc server address
	if len(viper.GetString("nkn_seedrpcserver_address")) > 0 {
		c.NKNSeedRPCServerAddress = nkn.NewStringArray(viper.GetString("nkn_seedrpcserver_address"))
	}

	// if `nkn_remote_peer` is missing in config file.
	if len(viper.GetString("nkn_remote_peer")) == 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("NKN address of remote peer: ")
		remote_peer_addr, _ := reader.ReadString('\n')
		remote_peer_addr = strings.TrimSuffix(remote_peer_addr, "\n")
		if len(remote_peer_addr) == 0 {
			return nil, errors.New("No remote peer address provided. Aborting.")
		}
		viper.Set("nkn_remote_peer", remote_peer_addr)
	}

	return c, nil
}

func (c *Config) Set(key string, value interface{}) {
	viper.Set(key, value)
}

func (c *Config) write() error {
	return viper.WriteConfig()
}
