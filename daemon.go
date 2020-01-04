package main

import (
	"fmt"
	"github.com/project-nano/framework"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"errors"
	"os"
	"github.com/project-nano/sonar"
	"log"
	"os/exec"
	"net"
	"github.com/project-nano/cell/service"
	"github.com/vishvananda/netlink"
	"bufio"
	"strings"
)
type DomainConfig struct {
	Domain        string `json:"domain"`
	GroupAddress  string `json:"group_address"`
	GroupPort     int    `json:"group_port"`
}

type MainService struct {
	cell *CellService
}

const (
	ExecuteName          = "cell"
	DomainConfigFileName = "domain.cfg"
	ConfigPathName       = "config"
	DataPathName         = "data"
	DefaultPathPerm      = 0740
	DefaultConfigPerm    = 0640
)

func (service *MainService)Start() (output string, err error){
	if nil == service.cell {
		err = errors.New("invalid service")
		return
	}
	if err = service.cell.Start();err != nil{
		return
	}
	output = fmt.Sprintf("\nCell Modeul %s\nservice %s listen at '%s:%d'\ngroup '%s:%d', domain '%s'",
		service.cell.GetVersion(),
		service.cell.GetName(), service.cell.GetListenAddress(), service.cell.GetListenPort(),
		service.cell.GetGroupAddress(), service.cell.GetGroupPort(), service.cell.GetDomain())
	return
}

func (service *MainService)Stop() (output string, err error){
	if nil == service.cell {
		err = errors.New("invalid service")
		return
	}
	err = service.cell.Stop()
	return
}

func generateConfigure(workingPath string) (err error){
	if err = configureNetworkForCell(); err != nil{
		fmt.Printf("configure cell network fail: %s\n", err.Error())
		return 
	}
	if err = checkDefaultRoute(); err != nil{
		fmt.Printf("check default route fail: %s\n", err.Error())
		return
	}
	var configPath = filepath.Join(workingPath, ConfigPathName)
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		//create path
		err = os.Mkdir(configPath, DefaultPathPerm)
		if err != nil {
			return
		}
		fmt.Printf("config path %s created\n", configPath)
	}

	var configFile = filepath.Join(configPath, DomainConfigFileName)
	if _, err = os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("No configures available, following instructions to generate a new one.")

		var config = DomainConfig{}
		if config.Domain, err = framework.InputString("Group Domain Name", sonar.DefaultDomain); err != nil{
			return
		}
		if config.GroupAddress, err = framework.InputString("Group MultiCast Address", sonar.DefaultMulticastAddress); err != nil{
			return
		}
		if config.GroupPort, err = framework.InputInteger("Group MultiCast Port", sonar.DefaultMulticastPort);err !=nil{
			return
		}
		//write
		data, err := json.MarshalIndent(config, "", " ")
		if err != nil {
			return err
		}
		if err = ioutil.WriteFile(configFile, data, DefaultConfigPerm); err != nil {
			return err
		}
		fmt.Printf("default configure '%s' generated\n", configFile)
	}
	return
}

func createDaemon(workingPath string) (daemon framework.DaemonizedService, err error){
	var configPath = filepath.Join(workingPath, ConfigPathName)
	var configFile = filepath.Join(configPath, DomainConfigFileName)
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}
	var config DomainConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return
	}
	var dataPath = filepath.Join(workingPath, DataPathName)
	if _, err = os.Stat(dataPath);os.IsNotExist(err){
		if err = os.Mkdir(dataPath, DefaultPathPerm);err != nil{
			return
		}else{
			log.Printf("data path '%s' created", dataPath)
		}
	}
	var inf *net.Interface
	inf, err = net.InterfaceByName(service.DefaultBridgeName)
	if err != nil{
		return
	}

	endpoint, err := framework.CreatePeerEndpoint(config.GroupAddress, config.GroupPort, config.Domain)
	if err != nil {
		return
	}
	var s = MainService{}
	s.cell = &CellService{EndpointService:endpoint, ConfigPath:configPath, DataPath:dataPath}
	s.cell.RegisterHandler(s.cell)
	err = s.cell.GenerateName(framework.ServiceTypeCell, inf)
	return &s, err
}

func checkDefaultRoute() (err error){
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil{
		return
	}
	if 0 == len(routes){
		err = errors.New("no route available")
		return
	}
	var defaultRouteAvailable = false
	for _, route := range routes{
		if route.Dst == nil{
			defaultRouteAvailable = true
		}
	}
	if !defaultRouteAvailable{
		err = errors.New("no default route available")
		return
	}
	fmt.Printf("default route ready\n")
	return nil
}

func configureNetworkForCell() (err error) {
	if hasDefaultBridge(){
		fmt.Printf("bridge %s is ready\n", service.DefaultBridgeName)
		return nil
	}
	ename, err := framework.SelectEthernetInterface("interface to bridge", true)
	if err != nil{
		return
	}
	fmt.Printf("try link interface '%s' to bridge '%s', input 'yes' to confirm:", ename, service.DefaultBridgeName)
	var input string
	_, err = fmt.Scanln(&input)
	if err != nil{
		return
	}
	if "yes" != input{
		return errors.New("user interrupted")
	}
	if err = linkBridge(ename, service.DefaultBridgeName);err != nil{
		return
	}

	{
		//disable & stop network manager
		var cmd = exec.Command("systemctl", "stop", "NetworkManager")
		if err = cmd.Run();err != nil{
			fmt.Printf("warning: stop networkmanager fail: %s", err.Error())
		}else{
			fmt.Println("network manager stopped")
		}
		cmd = exec.Command("systemctl", "disable", "NetworkManager")
		if err = cmd.Run();err != nil{
			fmt.Printf("warning: disable networkmanager fail: %s", err.Error())
		}else{
			fmt.Println("network manager disabled")
		}
	}
	{
		//restart network
		var cmd = exec.Command("systemctl", "stop", "network")
		if err = cmd.Run();err != nil{
			fmt.Printf("warning: stop network service fail: %s", err.Error())
		}else{
			fmt.Println("network service stopped")
		}
		cmd = exec.Command("systemctl", "start", "network")
		if err = cmd.Run();err != nil{
			fmt.Printf("warning: start network service fail: %s", err.Error())
			return
		}else{
			fmt.Println("network service restarted")
		}
	}
	return
}

func hasDefaultBridge() bool{
	list, err := net.Interfaces()
	if err != nil{
		fmt.Printf("fetch interface fail: %s", err.Error())
		return false
	}
	for _, i := range list{
		if service.DefaultBridgeName == i.Name{
			return true
		}
	}
	return false
}

func linkBridge(interfaceName, bridgeName string) (err error){
	const (
		ScriptsPath = "/etc/sysconfig/network-scripts"
		ScriptPrefix = "ifcfg"
	)
	var interfaceScript = filepath.Join(ScriptsPath, fmt.Sprintf("%s-%s", ScriptPrefix, interfaceName))
	var bridgeScript = filepath.Join(ScriptsPath, fmt.Sprintf("%s-%s", ScriptPrefix, bridgeName))
	interfaceConfig, err := readInterfaceConfig(interfaceScript)
	if err != nil{
		return
	}
	bridgeConfig, err := generateBridgeConfig(bridgeName)
	if err != nil{
		return
	}
	err = migrateInterfaceConfig(bridgeName, &interfaceConfig, &bridgeConfig)
	if err != nil{
		return
	}
	err = writeInterfaceConfig(interfaceConfig, interfaceScript)
	if err != nil{
		return
	}
	fmt.Printf("interface script %s updated\n", interfaceScript)
	err = writeInterfaceConfig(bridgeConfig, bridgeScript)
	if err != nil{
		return
	}
	fmt.Printf("bridge script %s generated\n", bridgeScript)
	link, err := netlink.LinkByName(interfaceName)
	if err != nil{
		return
	}
	if err = netlink.LinkSetDown(link);err != nil{
		fmt.Printf("warning:set down link fail: %s\n", err.Error())
	}
	var bridgeAttrs = netlink.NewLinkAttrs()
	bridgeAttrs.Name = bridgeName
	var bridge = &netlink.Bridge{LinkAttrs: bridgeAttrs}
	if err = netlink.LinkAdd(bridge);err != nil{
		return
	}
	fmt.Printf("new bridge %s created\n", bridgeName)
	if err = netlink.LinkSetMaster(link, bridge);err != nil{
		return
	}
	fmt.Printf("link %s added to bridge %s\n", interfaceName, bridgeName)
	if err = netlink.LinkSetUp(bridge); err != nil{
		return
	}
	fmt.Printf("bridge %s up\n", bridgeName)
	if err = netlink.LinkSetUp(link); err != nil{
		return
	}
	fmt.Printf("link %s up\n", interfaceName)
	return nil
}


type InterfaceConfig struct {
	Params map[string]string
}

func generateBridgeConfig(bridgeName string)(config InterfaceConfig, err error){
	config.Params = map[string]string{
		"NM_CONTROLLED": "no",
		"DELAY": "0",
		"TYPE": "Bridge",
		"ONBOOT": "yes",
		"ZONE": "public",
	}
	config.Params["NAME"] = bridgeName
	config.Params["DEVICE"] = bridgeName
	return config, nil
}
func readInterfaceConfig(filepath string) (config InterfaceConfig, err error){
	const (
		ValidDataCount = 2
		DataName = 0
		DataValue = 1
	)
	file, err := os.Open(filepath)
	if err != nil{
		return
	}
	config.Params = map[string]string{}
	var scanner = bufio.NewScanner(file)
	var lineIndex = 0
	for scanner.Scan(){
		var line = scanner.Text()
		var data = strings.Split(line, "=")
		lineIndex++
		if ValidDataCount != len(data){
			fmt.Printf("ignore line %d of '%s': %s\n", lineIndex, filepath, line)
			continue
		}
		config.Params[data[DataName]] = data[DataValue]
	}
	fmt.Printf("%d params loaded from '%s'\n", len(config.Params), filepath)
	return config, nil
}

func writeInterfaceConfig(config InterfaceConfig, filepath string) (err error){
	file, err := os.Create(filepath)
	if err != nil{
		return err
	}
	for name, value := range config.Params{
		fmt.Fprintf(file, "%s=%s\n", name, value)
	}
	return file.Close()
}

func migrateInterfaceConfig(bridgeName string, ifcfg, brcfg *InterfaceConfig) (err error){
	const (
		NMControl = "NM_CONTROLLED"
		BRIDGE    = "BRIDGE"
		ONBOOT    = "ONBOOT"
	)
	var migrateList = []string{
		"BOOTPROTO", "PREFIX", "IPADDR", "GATEWAY", "NETMASK", "DNS1", "DNS2", "DOMAIN",
		"DEFROUTE", "PEERDNS", "PEERROUTES", "IPV4_FAILURE_FATAL", "IPV6_FAILURE_FATAL", "PROXY_METHOD",
		"IPV6ADDR", "IPV6_DEFAULTGW", "IPV6_AUTOCONF", "IPV6_DEFROUTE", "IPV6INIT", "IPV6_ADDR_GEN_MODE",
	}

	for _, name := range migrateList{
		if value, exists := ifcfg.Params[name]; exists{
			brcfg.Params[name] = value
			delete(ifcfg.Params, name)
		}
	}
	ifcfg.Params[NMControl] = "no"
	ifcfg.Params[BRIDGE] = bridgeName
	ifcfg.Params[ONBOOT] = "yes"
	return nil
}

func main() {
	framework.ProcessDaemon(ExecuteName, generateConfigure, createDaemon)
}
