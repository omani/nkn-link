package config

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/nknorg/nkn-sdk-go"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const SEEDRPCSERVERADDR = "http://178.128.136.86:30003"
const IDENTIFIER = "nkn-link"
const DefaultMTU = 1420

type Config struct {
	path string

	DefaultRouteEnable         bool   `yaml:"default_route_enable"`
	DefaultRouteGatewayAddress string `yaml:"default_route_gateway_address"`
	EnableIPForwarding         bool   `yaml:"enable_ip_forwarding"`
	NKNAccountSeed             string `yaml:"nkn_account_seed"`
	NKNRemotePeer              string `yaml:"nkn_remote_peer"`
	NKNSeedRPCServerAddress    string `yaml:"nkn_seedrpcserver_address"`
	TunDeviceIPAddress         string `yaml:"tun_device_ip_address"`
	TunDeviceName              string `yaml:"tun_device_name"`
}

func NewConfig(path string) (*Config, error) {
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

			client, err := nkn.NewMultiClient(account, IDENTIFIER, 1, true, nil)
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
			os.Exit(0)
		}

		c := &Config{
			path: path,
		}

		err = yaml.Unmarshal(b, c)
		if err != nil {
			return nil, err
		}

		return c, nil
	}

	c := &Config{
		path: path,
	}

	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	// user preferred seed rpc server address
	if len(viper.GetString("nkn_seedrpcserver_address")) > 0 {
		c.NKNSeedRPCServerAddress = viper.GetString("nkn_seedrpcserver_address")
	}

	// if `nkn_remote_peer` is missing in config file.
	if len(c.NKNRemotePeer) == 0 {
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
	write()
}

func write() error {
	return viper.WriteConfig()
}

func (c *Config) NewAccount(seed []byte) (*nkn.Account, error) {
	return nkn.NewAccount(seed)
}

func (c *Config) NewMultiClient(acc *nkn.Account, id string, n int, orig bool) (*nkn.MultiClient, error) {
	return nkn.NewMultiClient(acc, id, n, orig, nil)
}

func (c *Config) GetNKNRemotePeer() *nkn.StringArray {
	return nkn.NewStringArray(c.NKNRemotePeer)
}

func (c *Config) GetNKNSeedRPCServerAddress() *nkn.StringArray {
	return nkn.NewStringArray(c.NKNSeedRPCServerAddress)
}
