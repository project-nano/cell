package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/project-nano/framework"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type InstanceNetworkResource struct {
	MonitorPort     int    `json:"monitor_port"`
	HardwareAddress string `json:"hardware_address,omitmepty"`
	InternalAddress string `json:"internal_address,omitmepty"`
	ExternalAddress string `json:"external_address,omitmepty"`
}

type networkCommand struct {
	Type         networkCommandType
	ResultChan   chan NetworkResult
	ErrorChan    chan error
	Instance     string
	InstanceList []string
	HWAddress    string
	Internal     string
	External     string
	Gateway      string
	DNS          []string
	Allocation   string
	Resources    map[string]InstanceNetworkResource
}

type NetworkManager struct {
	defaultBridge     string
	monitorPorts      map[int]bool
	instanceResources map[string]InstanceNetworkResource
	hwaddressMap      map[string]string //MAC => instance ID
	DHCPGateway       string
	DHCPDNS           []string
	AllocationMode    string
	commands          chan networkCommand
	dataFile          string
	generator         *rand.Rand
	util              *NetworkUtility
	OnAddressUpdated  func(string, []string)
	runner            *framework.SimpleRunner
}

type networkCommandType int

const (
	networkCommandGetCurrentConfig = iota
	networkCommandAllocateInstanceResource
	networkCommandDeallocateAllResource
	networkCommandAttachInstance
	networkCommandDetachInstance
	networkCommandUpdateAllocation
	networkCommandGetAddress
)

const (
	MonitorPortRangeBegin = 5901
	MonitorPortRange = 100
	MonitorPortRangeEnd = MonitorPortRangeBegin + MonitorPortRange
)


const (
	AddressAllocationNone      = ""
	AddressAllocationDHCP      = "dhcp"
	AddressAllocationCloudInit = "cloudinit"
)

func CreateNetworkManager(dataPath string, connect *libvirt.Connect) (*NetworkManager, error){
	const (
		NetworkFilename = "network.data"
		DefaultQueueSize = 1 << 10
	)
	var manager = NetworkManager{}
	manager.commands = make(chan networkCommand, DefaultQueueSize)
	manager.dataFile = filepath.Join(dataPath, NetworkFilename)
	var err error
	manager.runner = framework.CreateSimpleRunner(manager.Routine)

	manager.instanceResources = map[string]InstanceNetworkResource{}
	manager.hwaddressMap = map[string]string{}
	manager.monitorPorts = map[int]bool{}
	manager.generator = rand.New(rand.NewSource(time.Now().UnixNano()))
	manager.util = &NetworkUtility{connect}
	if changed, err := manager.util.DisableDHCPonDefaultNetwork(); err != nil{
		return nil, err
	}else if !changed{
		log.Println("<network> check pass: no dnsmasq DHCP available on default network")
	}else{
		log.Println("<network> dnsmasq DHCP disabled on default network")
	}
	if err = manager.loadConfig(); err != nil{
		return nil, err
	}
	return &manager, nil
}

func (manager *NetworkManager) SyncInstanceResources(resources map[string]InstanceNetworkResource) (err error){
	var modified = false
	for instanceID, resource := range resources{
		current, exists := manager.instanceResources[instanceID]
		if !exists{
			//new resource
			log.Printf("<network> add new resource of instance '%s'(%s), monitor %d, internal %s",
				instanceID, resource.HardwareAddress, resource.MonitorPort, resource.InternalAddress)
			modified = true
		}else{
			if current.MonitorPort != resource.MonitorPort{
				//port changed
				log.Printf("<network> sync monitor port of instance '%s' from %d => %d", instanceID, current.MonitorPort, resource.MonitorPort)
				manager.monitorPorts[current.MonitorPort] = false
				modified = true
			}
			if current.HardwareAddress != resource.HardwareAddress{
				log.Printf("<network> sync HW address of instance '%s' from %s => %s", instanceID, current.HardwareAddress, resource.HardwareAddress)
				modified = true
			}
		}
		manager.monitorPorts[resource.MonitorPort] = true
		manager.hwaddressMap[resource.HardwareAddress] = instanceID
	}
	if modified{
		return manager.saveConfig()
	}else{
		log.Println("<network> instance resource synchronized")
		return nil
	}
}

func (manager *NetworkManager) GetBridgeName() string{
	return DefaultBridgeName
}

func (manager *NetworkManager) GetCurrentConfig(resp chan NetworkResult){
	cmd := networkCommand{Type: networkCommandGetCurrentConfig, ResultChan:resp}
	manager.commands <- cmd
}

func (manager *NetworkManager) AllocateInstanceResource(instance, hwaddress, internal, external string, resp chan NetworkResult){
	cmd := networkCommand{Type: networkCommandAllocateInstanceResource, Instance:instance, HWAddress:hwaddress, Internal:internal, External:external, ResultChan:resp}
	manager.commands <- cmd
}
func (manager *NetworkManager)DeallocateAllResource(instance string, resp chan error){
	cmd := networkCommand{Type: networkCommandDeallocateAllResource, Instance:instance, ErrorChan:resp}
	manager.commands <- cmd
}

func (manager *NetworkManager)AttachInstances(resources map[string]InstanceNetworkResource, resp chan NetworkResult){
	manager.commands <- networkCommand{Type:networkCommandAttachInstance, Resources:resources, ResultChan:resp}
}

func (manager *NetworkManager)DetachInstances(instances []string, resp chan error){
	manager.commands <- networkCommand{Type:networkCommandDetachInstance, InstanceList: instances, ErrorChan:resp}
}

func (manager *NetworkManager) UpdateAddressAllocation(gateway string, dns []string, mode string, resp chan error){
	manager.commands <- networkCommand{Type: networkCommandUpdateAllocation, Gateway:gateway, DNS:dns, Allocation: mode, ErrorChan:resp}
}

func (manager *NetworkManager) GetAddressByHWAddress(hwaddress string, resp chan NetworkResult){
	manager.commands <- networkCommand{Type:networkCommandGetAddress, HWAddress:hwaddress, ResultChan:resp}
}

func (manager *NetworkManager) Start() error{
	return manager.runner.Start()
}

func (manager *NetworkManager) Stop() error{
	return manager.runner.Stop()
}

func (manager *NetworkManager)Routine(c framework.RoutineController){
	log.Println("<network> started")
	for !c.IsStopping(){
		select {
		case <- c.GetNotifyChannel():
			c.SetStopping()
			break
		case cmd := <- manager.commands:
			manager.handleCommand(cmd)
		}
	}
	c.NotifyExit()
	log.Println("<network> stopped")
}

func (manager *NetworkManager)handleCommand(cmd networkCommand){
	var err error
	switch cmd.Type {
	case networkCommandGetCurrentConfig:
		err = manager.handleGetCurrentConfig(cmd.ResultChan)
	case networkCommandAllocateInstanceResource:
		err = manager.handleAllocateInstanceResource(cmd.Instance, cmd.HWAddress, cmd.Internal, cmd.External, cmd.ResultChan)
	case networkCommandDeallocateAllResource:
		err = manager.handleDeallocateAllResource(cmd.Instance, cmd.ErrorChan)
	case networkCommandAttachInstance:
		err = manager.handleAttachInstances(cmd.Resources, cmd.ResultChan)
	case networkCommandDetachInstance:
		err = manager.handleDetachInstances(cmd.InstanceList, cmd.ErrorChan)
	case networkCommandUpdateAllocation:
		err = manager.handleUpdateAddressAllocation(cmd.Gateway, cmd.DNS, cmd.Allocation, cmd.ErrorChan)
	case networkCommandGetAddress:
		err = manager.handleGetAddressByHWAddress(cmd.HWAddress, cmd.ResultChan)
	default:
		log.Printf("<network> unsupported netword command %d", cmd.Type)
	}
	if err != nil{
		log.Printf("<network> handle command fail: %s", err.Error())
	}
}

func (manager *NetworkManager) handleGetCurrentConfig(respChan chan NetworkResult) error{
	if "" == manager.defaultBridge {
		return errors.New("no bridge available")
	}
	var result = NetworkResult{}
	result.Name = manager.defaultBridge
	result.Gateway = manager.DHCPGateway
	result.DNS = manager.DHCPDNS
	result.Allocation = manager.AllocationMode
	respChan <- result
	return nil
}

type networkDataConfig struct {
	DefaultBridge  string                             `json:"default_bridge"`
	Resources      map[string]InstanceNetworkResource `json:"resources,omitempty"`
	DNS            []string                           `json:"dns,omitempty"`
	Gateway        string                             `json:"gateway,omitempty"`
	AllocationMode string                             `json:"allocation_mode,omitempty"`
}

func (manager *NetworkManager)saveConfig() error{
	var config = networkDataConfig{}
	config.DefaultBridge = manager.defaultBridge
	config.Resources = manager.instanceResources
	config.DNS = manager.DHCPDNS
	config.Gateway = manager.DHCPGateway
	config.AllocationMode = manager.AllocationMode
	data, err := json.MarshalIndent(config, "", " ")
	if err != nil{
		return err
	}
	if err = ioutil.WriteFile(manager.dataFile, data, ConfigFilePerm); err != nil{
		return err
	}

	log.Printf("<network> %d resource groups saved to '%s'",
		len(config.Resources), manager.dataFile)
	return nil
}

func (manager *NetworkManager)loadConfig() error{
	//initial monitor pool
	for port := MonitorPortRangeBegin; port < MonitorPortRangeEnd; port++{
		manager.monitorPorts[port] = false
	}
	if _, err := os.Stat(manager.dataFile); os.IsNotExist(err){
		manager.defaultBridge = DefaultBridgeName
		log.Printf("<network> no config available, using default bridge '%s'", manager.defaultBridge)
		return manager.saveConfig()
	}
	var config networkDataConfig
	data, err := ioutil.ReadFile(manager.dataFile)
	if err != nil{
		return err
	}
	if err := json.Unmarshal(data, &config);err!= nil{
		return err
	}
	manager.defaultBridge = config.DefaultBridge
	manager.DHCPGateway = config.Gateway
	manager.DHCPDNS = config.DNS
	manager.AllocationMode = config.AllocationMode
	if config.Resources != nil{
		manager.instanceResources = config.Resources
	}

	var allocatedMonitorPortCount = 0
	for instanceID, instance := range manager.instanceResources{
		manager.monitorPorts[instance.MonitorPort] = true
		manager.hwaddressMap[instance.HardwareAddress] = instanceID
		allocatedMonitorPortCount++
	}
	log.Printf("<network> default bridge '%s' loaded", manager.defaultBridge)
	log.Printf("<network> %d / %d monitor port allocated, port range %d ~ %d",
		allocatedMonitorPortCount, len(manager.monitorPorts), MonitorPortRangeBegin, MonitorPortRangeEnd)
	return nil
}

func (manager *NetworkManager) handleAllocateInstanceResource(instance, hwaddress, internal, external string, resp chan NetworkResult) (err error){
	_, exists := manager.instanceResources[instance]
	if exists{
		err := fmt.Errorf("resource already allocated for instance '%s'", instance)
		resp <- NetworkResult{Error:err}
		return err
	}
	{
		//check format
		_, err = net.ParseMAC(hwaddress)
		if err != nil{
			err = fmt.Errorf("verify MAC '%s' fail: %s", hwaddress, err.Error())
			resp <- NetworkResult{Error:err}
			return err
		}
		if "" != internal{
			_, _, err = net.ParseCIDR(internal)
			if err != nil{
				err = fmt.Errorf("verify internal address '%s' fail: %s", internal, err.Error())
				resp <- NetworkResult{Error:err}
				return err
			}
		}
		if "" != external{
			_, _, err = net.ParseCIDR(external)
			if err != nil{
				err = fmt.Errorf("verify external address '%s' fail: %s", external, err.Error())
				resp <- NetworkResult{Error:err}
				return err
			}
		}
	}
	var selected = 0
	var seed = manager.generator.Intn(MonitorPortRange)
	var offset = 0
	for ; offset < MonitorPortRange; offset++{
		var port = MonitorPortRangeBegin + (seed + offset) % MonitorPortRange
		allocated, exists := manager.monitorPorts[port]
		if !exists{
			err := fmt.Errorf("encounter invalid monitor port %d", port)
			resp <- NetworkResult{Error:err}
			return err
		}
		if allocated{
			continue
		}
		selected = port
		manager.monitorPorts[selected] = true
		break
	}
	if 0 == selected{
		err := errors.New("no port available")
		resp <- NetworkResult{Error:err}
		return err
	}
	log.Printf("<network> monitor port %d allocated for instance '%s'", selected, instance)
	manager.instanceResources[instance] = InstanceNetworkResource{selected, hwaddress, internal, external}
	manager.hwaddressMap[hwaddress] = instance
	resp <- NetworkResult{MonitorPort:selected}
	return manager.saveConfig()
}

func (manager *NetworkManager) handleDeallocateAllResource(instance string, resp chan error) error{
	resource, exists := manager.instanceResources[instance]
	if !exists{
		err := fmt.Errorf("no resource allocated for instance '%s'", instance)
		resp <- err
		return err
	}

	{
		//monitor port
		allocated, exists := manager.monitorPorts[resource.MonitorPort]
		if !exists{
			err := fmt.Errorf("invalid monitor port %d for instance '%s'", resource.MonitorPort, instance)
			resp <- err
			return err
		}
		if !allocated{
			err := fmt.Errorf("target monitor port %d not allocated", resource.MonitorPort)
			resp <- err
			return err
		}
		//deallocate
		manager.monitorPorts[resource.MonitorPort] = false
		log.Printf("<network> monitor port %d deallocated", resource.MonitorPort)
	}
	{
		//HW address
		boundInstance, exists := manager.hwaddressMap[resource.HardwareAddress]
		if !exists{
			var err = fmt.Errorf("invalid hardware address '%s' for instance '%s'", resource.HardwareAddress, instance)
			resp <- err
			return err
		}
		if boundInstance != instance{
			var err = fmt.Errorf("deallocate instance '%s', but MAC '%s' already bound with '%s'",
				instance, resource.HardwareAddress, boundInstance)
			resp <- err
			return err
		}
		delete(manager.hwaddressMap, resource.HardwareAddress)
		log.Printf("<network> resource on MAC '%s' released", resource.HardwareAddress)
	}
	delete(manager.instanceResources, instance)
	log.Printf("<network> resource deallocated for instance '%s'", instance)
	resp <- nil
	return manager.saveConfig()
}

func (manager *NetworkManager) handleAttachInstances(allocatedResources map[string]InstanceNetworkResource, respChan chan NetworkResult) (err error){
	var instances []string
	for instanceID, _ := range allocatedResources {
		if _, exists := manager.instanceResources[instanceID];exists{
			err = fmt.Errorf("instance '%s' already exists", instanceID)
			respChan <- NetworkResult{Error:err}
			return err
		}
		instances = append(instances, instanceID)
	}
	var required = len(allocatedResources)
	var result = map[string]InstanceNetworkResource{}
	var selected = 0
	var seed = manager.generator.Intn(MonitorPortRange)
	var offset = 0
	for ; offset < MonitorPortRange; offset++ {
		var port = MonitorPortRangeBegin + (seed+offset)%MonitorPortRange
		//log.Printf("<network> debug: seed %d, offset %d, port %d", seed, offset, port)
		portAllocated, exists := manager.monitorPorts[port]
		if !exists{
			err = fmt.Errorf("invalid monitor port %d generated", port)
			respChan <- NetworkResult{Error:err}
			return err
		}
		if portAllocated {
			continue
		}
		manager.monitorPorts[port] = true
		var instanceID = instances[selected]
		current, exists := allocatedResources[instanceID]
		if !exists{
			err = fmt.Errorf("no allocated resource for instance %s", instanceID)
			respChan <- NetworkResult{Error:err}
			return err
		}
		result[instanceID] = InstanceNetworkResource{port, current.HardwareAddress, current.InternalAddress, current.ExternalAddress}
		log.Printf("<network> attach monitor port %d for instance '%s'", port, instanceID)
		selected++
		if selected >= required{
			break
		}
	}
	if selected < required{
		err = fmt.Errorf("only %d port available for %d instance(s)", selected, required)
		//release
		for _, resource := range result {
			manager.monitorPorts[resource.MonitorPort] = false
		}
		respChan <- NetworkResult{Error:err}
		return err
	}
	//attach
	for instanceID, resource := range result {
		manager.instanceResources[instanceID] = resource
		manager.hwaddressMap[resource.HardwareAddress] = instanceID
	}
	respChan <- NetworkResult{Resources: result}
	return manager.saveConfig()
}

func (manager *NetworkManager) handleDetachInstances(instances []string, respChan chan error) (err error){
	if 0 == len(instances){
		for id, _ := range manager.instanceResources{
			instances = append(instances, id)
		}
	}
	for _, instanceID := range instances{
		resource, exists := manager.instanceResources[instanceID]
		if !exists{
			err = fmt.Errorf("no resource available for instance '%s'", instanceID)
			respChan <- err
			return err
		}
		allocated, exists := manager.monitorPorts[resource.MonitorPort]
		if !exists{
			err = fmt.Errorf("invalid moinitor port %d for instance '%s'", resource.MonitorPort, instanceID)
			respChan <- err
			return err
		}
		if !allocated{
			err = fmt.Errorf("moinitor port %d not allcated for instance '%s'", resource.MonitorPort, instanceID)
			respChan <- err
			return err
		}
		manager.monitorPorts[resource.MonitorPort] = false
		delete(manager.instanceResources, instanceID)
		log.Printf("<network> detach monitor port %d for instance '%s'", resource.MonitorPort, instanceID)
	}
	respChan <- nil
	return manager.saveConfig()
}


func (manager *NetworkManager) handleUpdateAddressAllocation(gateway string, dns []string, allocationMode string, respChan chan error) (err error){
	if !isValidIPv4(gateway){
		err = fmt.Errorf("invalid gateway '%s'", gateway)
		respChan <- err
		return
	}
	for _, server := range dns{
		if !isValidIPv4(server){
			err = fmt.Errorf("invalid DNS server '%s'", server)
			respChan <- err
			return
		}
	}
	manager.DHCPDNS = dns
	manager.DHCPGateway = gateway
	manager.AllocationMode = allocationMode
	log.Printf("<network> address allocation updated to mode '%s', gateway '%s', DNS: %s",
		allocationMode, gateway, strings.Join(dns, "/"))
	respChan <- nil
	if AddressAllocationDHCP == allocationMode &&manager.OnAddressUpdated != nil{
		manager.OnAddressUpdated(gateway, dns)
	}
	return manager.saveConfig()
}

func (manager *NetworkManager) handleGetAddressByHWAddress(hwaddress string, respChan chan NetworkResult) (err error){
	instanceID, exists := manager.hwaddressMap[hwaddress]
	if !exists{
		err = fmt.Errorf("invalid HWAddress '%s'", hwaddress)
		respChan <- NetworkResult{Error:err}
		return
	}
	resource, exists := manager.instanceResources[instanceID]
	if !exists{
		err = fmt.Errorf("no network resource available for instance '%s'", instanceID)
		respChan <- NetworkResult{Error:err}
		return
	}
	if "" == resource.InternalAddress{
		err = fmt.Errorf("no internal address assigned for instance '%s'", instanceID)
		respChan <- NetworkResult{Error:err}
		return
	}
	var result NetworkResult
	result.Gateway = manager.DHCPGateway
	result.DNS = manager.DHCPDNS
	result.Internal = resource.InternalAddress
	result.External = resource.ExternalAddress
	log.Printf("<network> get internal address '%s' for MAC '%s'", resource.InternalAddress, hwaddress)
	respChan <- result
	return nil
}

func isValidIPv4(value string) bool {
	var ip = net.ParseIP(value)
	if ip == nil{
		return false
	}
	ip = ip.To4()
	return ip != nil
}