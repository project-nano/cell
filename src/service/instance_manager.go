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
	"os"
	"path/filepath"
	"time"
)

const (
	InstanceMediaOptionNone    uint = iota
	InstanceMediaOptionImage
	InstanceMediaOptionNetwork
)

type InstanceNetworkMode int

const (
	NetworkModePrivate = iota
	NetworkModePlain
	NetworkModeMono
	NetworkModeShare
	NetworkModeVPC
)

type PriorityEnum uint

const (
	PriorityHigh = iota
	PriorityMedium
	PriorityLow
)

type TemplateOperatingSystem int

const (
	TemplateOperatingSystemLinux = iota
	TemplateOperatingSystemWindows
	TemplateOperatingSystemInvalid
)

func (value TemplateOperatingSystem) ToString() string{
	switch value{
	case TemplateOperatingSystemLinux:
		return SystemNameLinux
	case TemplateOperatingSystemWindows:
		return SystemNameWindows
	default:
		return "invalid"
	}
}

type TemplateDiskDriver int

const (
	TemplateDiskDriverSCSI = iota
	TemplateDiskDriverSATA
	TemplateDiskDriverIDE
	TemplateDiskDriverInvalid
)

func (value TemplateDiskDriver) ToString() string{
	switch value{
	case TemplateDiskDriverSCSI:
		return DiskBusSCSI
	case TemplateDiskDriverSATA:
		return DiskBusSATA
	case TemplateDiskDriverIDE:
		return DiskBusIDE
	default:
		return "invalid"
	}
}

type TemplateNetworkModel int

const (
	TemplateNetworkModelVirtIO = iota
	TemplateNetworkModelE1000
	TemplateNetworkModelRTL18139
	TemplateNetworkModelInvalid
)

func (value TemplateNetworkModel) ToString() string{
	switch value{
	case TemplateNetworkModelVirtIO:
		return NetworkModelVIRTIO
	case TemplateNetworkModelE1000:
		return NetworkModelE1000
	case TemplateNetworkModelRTL18139:
		return NetworkModelRTL8139
	default:
		return "invalid"
	}
}

type TemplateDisplayDriver int

const (
	TemplateDisplayDriverVGA = iota
	TemplateDisplayDriverCirrus
	TemplateDisplayDriverVirtIO
	TemplateDisplayDriverQXL
	TemplateDisplayDriverNone
	TemplateDisplayDriverInvalid
)

func (value TemplateDisplayDriver) ToString() string{
	switch value{
	case TemplateDisplayDriverVGA:
		return DisplayDriverVGA
	case TemplateDisplayDriverCirrus:
		return DisplayDriverCirrus
	case TemplateDisplayDriverVirtIO:
		return DisplayDriverVirtIO
	case TemplateDisplayDriverQXL:
		return DisplayDriverQXL
	case TemplateDisplayDriverNone:
		return DisplayDriverNone
	default:
		return "invalid"
	}
}

type TemplateRemoteControl int

const (
	TemplateRemoteControlVNC = iota
	TemplateRemoteControlSPICE
	TemplateRemoteControlInvalid
)

func (value TemplateRemoteControl) ToString() string{
	switch value{
	case TemplateRemoteControlVNC:
		return RemoteControlVNC
	case TemplateRemoteControlSPICE:
		return RemoteControlSPICE
	default:
		return "invalid"
	}
}

type TemplateUSBModel int

const (
	TemplateUSBModelNone = iota
	TemplateUSBModelXHCI
	TemplateUSBModelInvalid
)

func (value TemplateUSBModel) ToString() string{
	switch value{
	case TemplateUSBModelNone:
		return USBModelNone
	case TemplateUSBModelXHCI:
		return USBModelXHCI
	default:
		return "invalid"
	}
}

type TemplateTabletModel int

const (
	TemplateTabletModelNone = iota
	TemplateTabletModelUSB
	TemplateTabletModelVirtIO
	TemplateTabletModelInvalid
)

func (value TemplateTabletModel) ToString() string{
	switch value{
	case TemplateTabletModelNone:
		return TabletBusNone
	case TemplateTabletModelUSB:
		return TabletBusUSB
	case TemplateTabletModelVirtIO:
		return TabletBusVIRTIO
	default:
		return "invalid"
	}
}

type HardwareTemplate struct {
	OperatingSystem string `json:"operating_system"`
	Disk            string `json:"disk"`
	Network         string `json:"network"`
	Display         string `json:"display"`
	Control         string `json:"control"`
	USB             string `json:"usb,omitempty"`
	Tablet          string `json:"tablet,omitempty"`
}


type GuestConfig struct {
	Name               string              `json:"name"`
	ID                 string              `json:"id"`
	User               string              `json:"user"`
	Group              string              `json:"group"`
	Cores              uint                `json:"cores"`
	Memory             uint                `json:"memory"`
	Disks              []uint64            `json:"disks"`
	AutoStart          bool                `json:"auto_start"`
	System             string              `json:"system,omitempty"`
	MonitorPort        uint                `json:"monitor_port"`
	MonitorSecret      string              `json:"monitor_secret"`
	Created            bool                `json:"-"`
	Progress           uint                `json:"-"` //limit to 100
	Running            bool                `json:"-"`
	StorageMode        InstanceStorageMode `json:"storage_mode"`
	StoragePool        string              `json:"storage_pool"`
	StorageVolumes     []string            `json:"storage_volumes"`
	NetworkMode        InstanceNetworkMode `json:"network_mode"`
	NetworkSource      string              `json:"network_source"`
	NetworkAddress     string              `json:"-"`
	HardwareAddress    string              `json:"hardware_address"`
	Ports              []uint              `json:"ports,omitempty"`
	AuthUser           string              `json:"auth_user,omitempty"`
	AuthSecret         string              `json:"auth_secret,omitempty"`
	SystemVersion      string              `json:"system_version,omitempty"`
	Initialized        bool                `json:"initialized,omitempty"`
	RootLoginEnabled   bool                `json:"root_login_enabled,omitempty"`
	DataPath           string              `json:"data_path,omitempty"`
	QEMUAvailable      bool                `json:"qemu_available,omitempty"`
	CloudInitAvailable bool                `json:"cloud_init_available,omitempty"`
	BootImage          string              `json:"boot_image,omitempty"`
	CreateTime         string              `json:"create_time,omitempty"`
	AddressAllocation  string              `json:"address_allocation,omitempty"`
	InternalAddress    string              `json:"internal_address,omitemtpy"`
	ExternalAddress    string              `json:"external_address,omitemtpy"`
	CPUPriority        PriorityEnum        `json:"cpu_priority,omitempty"`
	WriteSpeed         uint64              `json:"write_speed,omitempty"`
	WriteIOPS          uint64              `json:"write_iops,omitempty"`
	ReadSpeed          uint64              `json:"read_speed,omitempty"`
	ReadIOPS           uint64              `json:"read_iops,omitempty"`
	ReceiveSpeed       uint64              `json:"receive_speed,omitempty"`
	SendSpeed          uint64              `json:"send_speed,omitempty"`
	Template           *HardwareTemplate   `json:"template,omitempty"`
}

type InstanceStatus struct {
	GuestConfig
	MediaAttached   bool
	MediaSource     string
	CpuUsages       float64
	AvailableMemory uint64
	AvailableDisk   uint64
	BytesRead       uint64
	BytesWritten    uint64
	BytesReceived   uint64
	BytesSent       uint64
	InstanceSnapshot
}

type InstanceSnapshot struct {
	lastCPUTimes uint64
	lastCPUCheck time.Time
	startTime    time.Time
}

const (
	InstanceStatusStopped = iota
	InstanceStatusRunning
)

type InstanceStorageMode int

const (
	StorageModeLocal = iota
	StorageModeNFS
)

type instanceCommand struct {
	Type            InstanceCommandType
	Instance        string
	URL             string
	Config          GuestConfig
	Reboot          bool
	Force           bool
	Host            string
	Port            uint
	Cores           uint
	Memory          uint
	Password        string
	User            string
	Index           int
	Size            uint64
	Name            string
	Allocation      string
	Media           InstanceMediaConfig
	Priority        PriorityEnum
	ReadSpeed       uint64
	WriteSpeed      uint64
	ReadIOPS        uint64
	WriteIOPS       uint64
	ReceiveSpeed    uint64
	SendSpeed       uint64
	InstanceList    []string
	NetworkResource map[string]InstanceNetworkResource
	ResultChan      chan InstanceResult
	ErrorChan       chan error
	AllConfigChan   chan []GuestConfig
	BoolChan        chan bool
	EventChan       chan InstanceStatusChangedEvent
}

type InstanceCommandType int

const (
	InsCmdCreate              = iota
	InsCmdDelete
	InsCmdGet
	InsCmdGetStatus
	InsCmdGetAllConfig
	InsCmdModify
	InsCmdStart
	InsCmdStop
	InsCmdAttach
	InsCmdDetach
	InsCmdIsRunning
	InsCmdModifyCore
	InsCmdModifyMemory
	InsCmdModifyAuth
	InsCmdGetAuth
	InsCmdUpdateDiskSize
	InsCmdAddEventListener
	InsCmdRemoveEventListener
	InsCmdFinishInitialize
	InsCmdInsertMedia
	InsCmdEjectMedia
	InsCmdUsingStorage
	InsCmdDetachStorage
	InsCmdGetNetworkResource
	InsCmdAttachInstance
	InsCmdDetachInstance
	InsCmdMigrateInstance
	InsCmdRename
	InsCmdResetSystem
	InsCmdModifyCPUPriority
	InsCmdModifyDiskThreshold
	InsCmdModifyNetworkThreshold
	InsCmdResetMonitorSecret
	InsCmdSyncAddressAllocation
	InsCmdInvalid
)

var instanceCommandNames = []string{
	"Create",
	"Delete",
	"Get",
	"GetStatus",
	"GetAllConfig",
	"Modify",
	"Start",
	"Stop",
	"Attach",
	"Detach",
	"IsRunning",
	"ModifyCore",
	"ModifyMemory",
	"ModifyAuth",
	"GetAuth",
	"UpdateDiskSize",
	"AddEventListener",
	"RemoveEventListener",
	"FinishInitialize",
	"InsertMedia",
	"EjectMedia",
	"UsingStorage",
	"DetachStorage",
	"GetNetworkResource",
	"AttachInstance",
	"DetachInstance",
	"MigrateInstance",
	"Rename",
	"ResetSystem",
	"ModifyCPUPriority",
	"ModifyDiskThreshold",
	"ModifyNetworkThreshold",
	"ResetMonitorSecret",
	"SyncAddressAllocation",
}

func (c InstanceCommandType) toString() string {
	if c >= InsCmdInvalid{
		return  "invalid"
	}
	return instanceCommandNames[c]
}

type InstanceStatusChangedEvent struct {
	ID        string
	Event     StatusChangedEvent
	Address   string
	Timestamp time.Time
}

type StatusChangedEvent int

const (
	InstanceStarted = iota
	InstanceStopped
	AddressChanged
	GuestCreated
	GuestDeleted
)

const (
	MediaImagePath    = "media_images"
	SyncInterval      = 2 * time.Second
	SystemNameLinux   = "linux"
	SystemNameWindows = "windows"
	AdminLinux        = "root"
	AdminWindows      = "Administrator"
	MetaFileSuffix    = "meta"
)

const (
	TimeFormatLayout = "2006-01-02 15:04:05"
)

type InstanceManager struct {
	defaultTemplate HardwareTemplate
	commands        chan instanceCommand
	dataFile        string
	instances       map[string]InstanceStatus
	util            *InstanceUtility
	storagePool     string
	storageURL      string
	events          chan InstanceStatusChangedEvent
	eventListeners  map[string]chan InstanceStatusChangedEvent
	randomGenerator *rand.Rand
	runner          *framework.SimpleRunner
}

func CreateInstanceManager(dataPath string, connect *libvirt.Connect) (manager *InstanceManager, err error) {
	if InsCmdInvalid != len(instanceCommandNames){
		err = fmt.Errorf("insufficient command names %d/%d", len(instanceCommandNames), InsCmdInvalid)
		return
	}

	const (
		InstanceFilename = "instance.data"
		DefaultQueueSize = 1 << 10
	)
	manager = &InstanceManager{}
	manager.runner = framework.CreateSimpleRunner(manager.Routine)
	manager.commands = make(chan instanceCommand, DefaultQueueSize)
	manager.events = make(chan InstanceStatusChangedEvent, DefaultQueueSize)
	manager.dataFile = filepath.Join(dataPath, InstanceFilename)
	manager.instances = map[string]InstanceStatus{}
	manager.eventListeners = map[string]chan InstanceStatusChangedEvent{}
	manager.defaultTemplate = HardwareTemplate{
		OperatingSystem: TemplateOperatingSystem(TemplateOperatingSystemLinux).ToString(),
		Disk:            TemplateDiskDriver(TemplateDiskDriverSCSI).ToString(),
		Network:         TemplateNetworkModel(TemplateNetworkModelVirtIO).ToString(),
		Display:         TemplateDisplayDriver(TemplateDisplayDriverVGA).ToString(),
		Control:         TemplateRemoteControl(TemplateRemoteControlVNC).ToString(),
		USB:             TemplateUSBModel(TemplateUSBModelNone).ToString(),
		Tablet:          TemplateTabletModel(TemplateTabletModelUSB).ToString(),
	}
	manager.randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
	if manager.util, err = CreateInstanceUtility(connect); err != nil {
		return nil, err
	}
	if err = manager.loadConfig(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (manager *InstanceManager) GetInstanceNetworkResources() (result map[string]InstanceNetworkResource){
	result = map[string]InstanceNetworkResource{}
	for instanceID, instance := range manager.instances{
		result[instanceID] = InstanceNetworkResource{
			MonitorPort: int(instance.MonitorPort),
			HardwareAddress: instance.HardwareAddress,
			InternalAddress: instance.InternalAddress,
			ExternalAddress: instance.ExternalAddress,
			}
	}
	return result
}

func (manager *InstanceManager) GetEventChannel() chan InstanceStatusChangedEvent {
	return manager.events
}

func (manager *InstanceManager) Start() error{
	return manager.runner.Start()
}

func (manager *InstanceManager) Stop() error{
	return manager.runner.Stop()
}

func (manager *InstanceManager) Routine(c framework.RoutineController) {
	log.Println("<instance> started")
	var syncTicker = time.NewTicker(SyncInterval)
	for !c.IsStopping() {
		select {
		case <-c.GetNotifyChannel():
			c.SetStopping()
			break
		case cmd := <-manager.commands:
			manager.handleCommand(cmd)
		case <-syncTicker.C:
			manager.syncInstanceStatus()
		}
	}
	c.NotifyExit()
	log.Println("<instance> stopped")
}

func (manager *InstanceManager) syncInstanceStatus() {
	const (
		MinimalCPUGap        = 1 * time.Second
		NetworkCheckInterval = 30 * time.Second
		NetworkCheckDivider  = int(NetworkCheckInterval / SyncInterval)
	)
	var now = time.Now()

	for id, status := range manager.instances {
		isRunning, err := manager.util.IsInstanceRunning(id)
		if err != nil {
			log.Printf("<instance> check instance status fail: %s", err.Error())
			continue
		}
		if isRunning != status.Running {
			if isRunning {
				//stopped => running

				log.Printf("<instance> sync instance '%s' status => running", id)
				manager.StartCPUMonitor(&status)
				manager.events <- InstanceStatusChangedEvent{ID: id, Event: InstanceStarted, Timestamp: time.Now()}
			} else {
				//running => stopped
				log.Printf("<instance> sync instance '%s' status => stopped", id)
				status.CpuUsages = 0.0
				manager.events <- InstanceStatusChangedEvent{ID: id, Event: InstanceStopped, Timestamp: time.Now()}
			}
			status.Running = isRunning
			manager.instances[id] = status
		} else if isRunning {
			//check usage
			if status.lastCPUCheck.Add(MinimalCPUGap).After(now) {
				//log.Printf("<instance> warning: CPU check ignore %s", id)
				continue
			}
			times, cores, err := manager.util.GetCPUTimes(id)
			if err != nil {
				log.Printf("<instance> get CPU times fail: %s", err.Error())
				continue
			}
			if cores != status.Cores {
				log.Printf("<instance> warning: cores of instance '%s' change from %d to %d", id, status.Cores, cores)
				continue
			}
			var usedNanoseconds = times - status.lastCPUTimes
			if usedNanoseconds < 0 {
				log.Printf("<instance> expected used CPU times %d for instance '%s'", usedNanoseconds, id)
				continue
			}

			var elapsedNanoseconds = now.Sub(status.lastCPUCheck).Nanoseconds()
			if usedNanoseconds < 0 {
				log.Printf("<instance> expected elapsed CPU times %d for instance '%s'", elapsedNanoseconds, id)
				continue
			}
			status.CpuUsages = float64(usedNanoseconds*100) / float64(elapsedNanoseconds*int64(status.Cores))
			//log.Printf("<instance> debug: instance %s => %.02f%%  ( %d / %d)", status.Name, status.CpuUsages, usedNanoseconds, elapsedNanoseconds)
			status.lastCPUTimes = times
			status.lastCPUCheck = now

			if status.NetworkAddress == "" {
				//check network interface
				var elapsed = int(now.Sub(status.startTime) / SyncInterval)
				if (NetworkCheckDivider - 1) == elapsed % NetworkCheckDivider {
					//get ip address
					ip, err := manager.util.GetIPv4Address(id, status.HardwareAddress)
					if (err == nil) && (ip != "") {
						status.NetworkAddress = ip
						log.Printf("<instance> ip '%s' detected for instance '%s'", ip, status.Name)
						manager.events <- InstanceStatusChangedEvent{ID: id, Event: AddressChanged, Address: ip, Timestamp: time.Now()}
					}
				}
			}else{
				//detect interval change to 2 min after established some IP
				var currentIP = status.NetworkAddress
				var elapsed = int(now.Sub(status.startTime) / SyncInterval)
				var prolongedDivider = NetworkCheckDivider * 4 // 30s => 2min
				if (prolongedDivider - 1) == elapsed % prolongedDivider {
					//get ip address
					ip, err := manager.util.GetIPv4Address(id, status.HardwareAddress)
					if (err == nil) && (ip != currentIP) {
						status.NetworkAddress = ip
						log.Printf("<instance> IP of instance '%s' changed from '%s' to '%s'", status.Name, currentIP, ip)
						manager.events <- InstanceStatusChangedEvent{ID: id, Event: AddressChanged, Address: ip, Timestamp: time.Now()}
					}
				}
			}

			manager.instances[id] = status
		}
	}
}

func (manager *InstanceManager) UsingStorage(name, url string, respChan chan error){
	manager.commands <- instanceCommand{Type:InsCmdUsingStorage, Name:name, URL:url, ErrorChan:respChan}
}

func (manager *InstanceManager) DetachStorage(respChan chan error){
	manager.commands <- instanceCommand{Type:InsCmdDetachStorage, ErrorChan:respChan}
}

func (manager *InstanceManager) CreateInstance(config GuestConfig, resp chan error) {
	cmd := instanceCommand{Type: InsCmdCreate, Config: config, ErrorChan: resp}
	manager.commands <- cmd
}

func (manager *InstanceManager) DeleteInstance(id string, resp chan error) {
	cmd := instanceCommand{Type: InsCmdDelete, Instance: id, ErrorChan: resp}
	manager.commands <- cmd
}

func (manager *InstanceManager) GetInstanceConfig(id string, resp chan InstanceResult) {
	cmd := instanceCommand{Type: InsCmdGet, Instance: id, ResultChan: resp}
	manager.commands <- cmd
}

func (manager *InstanceManager) GetInstanceStatus(id string, resp chan InstanceResult) {
	cmd := instanceCommand{Type: InsCmdGetStatus, Instance: id, ResultChan: resp}
	manager.commands <- cmd
}

func (manager *InstanceManager) StartInstance(id string, resp chan error) {
	cmd := instanceCommand{Type: InsCmdStart, Instance: id, ErrorChan: resp}
	manager.commands <- cmd
}
func (manager *InstanceManager) StartInstanceWithMedia(id string, media InstanceMediaConfig, resp chan error) {
	cmd := instanceCommand{Type: InsCmdStart, Instance: id, Media:media, ErrorChan: resp}
	manager.commands <- cmd
}
func (manager *InstanceManager) StopInstance(id string, reboot, force bool, resp chan error) {
	cmd := instanceCommand{Type: InsCmdStop, Instance: id, Reboot: reboot, Force: force, ErrorChan: resp}
	manager.commands <- cmd
}

func (manager *InstanceManager) GetAllInstance(resp chan []GuestConfig) {
	cmd := instanceCommand{Type: InsCmdGetAllConfig, AllConfigChan: resp}
	manager.commands <- cmd
}
func (manager *InstanceManager) IsInstanceRunning(id string, resp chan bool) {
	cmd := instanceCommand{Type: InsCmdIsRunning, Instance: id, BoolChan: resp}
	manager.commands <- cmd
}


func (manager *InstanceManager) ModifyGuestName(id, name string, resp chan error){
	manager.commands <- instanceCommand{Type:InsCmdRename, Instance:id, Name: name, ErrorChan:resp}
}

func (manager *InstanceManager) ModifyGuestCore(id string, core uint, resp chan error) {
	manager.commands <- instanceCommand{Type: InsCmdModifyCore, Instance: id, Cores: core, ErrorChan: resp}
}

func (manager *InstanceManager) ModifyGuestMemory(id string, memory uint, resp chan error) {
	manager.commands <- instanceCommand{Type: InsCmdModifyMemory, Instance: id, Memory: memory, ErrorChan: resp}
}

func (manager *InstanceManager) ModifyCPUPriority(guestID string, priority PriorityEnum, resp chan error){
	manager.commands <- instanceCommand{Type: InsCmdModifyCPUPriority, Instance: guestID, Priority: priority, ErrorChan: resp}
}

func (manager *InstanceManager) ModifyDiskThreshold(guestID string, readSpeed, readIOPS, writeSpeed, writeIOPS uint64, resp chan error){
	manager.commands <- instanceCommand{Type: InsCmdModifyDiskThreshold, Instance: guestID, ReadSpeed: readSpeed, ReadIOPS: readIOPS, WriteSpeed:writeSpeed, WriteIOPS: writeIOPS, ErrorChan: resp}
}

func (manager *InstanceManager) ModifyNetworkThreshold(guestID string, receive, send uint64, resp chan error){
	manager.commands <- instanceCommand{Type: InsCmdModifyNetworkThreshold, Instance: guestID, ReceiveSpeed:receive, SendSpeed: send, ErrorChan: resp}
}

func (manager *InstanceManager) ModifyGuestAuth(id, password, usr string, resp chan InstanceResult) {
	manager.commands <- instanceCommand{Type: InsCmdModifyAuth, Instance: id, Password: password, User: usr, ResultChan: resp}
}

func (manager *InstanceManager) GetGuestAuth(id string, resp chan InstanceResult) {
	manager.commands <- instanceCommand{Type: InsCmdGetAuth, Instance: id, ResultChan: resp}
}

func (manager *InstanceManager) FinishGuestInitialize(id string, resp chan error){
	manager.commands <- instanceCommand{Type: InsCmdFinishInitialize, Instance: id, ErrorChan:resp}
}
func (manager *InstanceManager) UpdateDiskSize(guest string, index int, size uint64, resp chan error) {
	manager.commands <- instanceCommand{Type: InsCmdUpdateDiskSize, Instance: guest, Index: index, Size: size, ErrorChan: resp}
}

func (manager *InstanceManager) AddEventListener(listener string, eventChan chan InstanceStatusChangedEvent){
	manager.commands <- instanceCommand{Type: InsCmdAddEventListener, Name:listener, EventChan:eventChan}
}

func (manager *InstanceManager) RemoveEventListener(listener string){
	manager.commands <- instanceCommand{Type: InsCmdRemoveEventListener, Name:listener}
}

func (manager *InstanceManager) AttachMedia(id string, media InstanceMediaConfig, resp chan error){
	manager.commands <- instanceCommand{Type: InsCmdInsertMedia, Instance:id, Media:media, ErrorChan:resp}
}

func (manager *InstanceManager) DetachMedia(id string, resp chan error){
	manager.commands <- instanceCommand{Type: InsCmdEjectMedia, Instance:id, ErrorChan:resp}
}

func (manager *InstanceManager) GetNetworkResources(instances []string, respChan chan InstanceResult){
	manager.commands <- instanceCommand{Type: InsCmdGetNetworkResource, InstanceList: instances, ResultChan:respChan}
}
func (manager *InstanceManager) AttachInstances(resources map[string]InstanceNetworkResource, respChan chan error){
	manager.commands <- instanceCommand{Type: InsCmdAttachInstance, NetworkResource: resources, ErrorChan:respChan}
}

func (manager *InstanceManager) DetachInstances(instances []string, respChan chan error){
	manager.commands <- instanceCommand{Type: InsCmdDetachInstance, InstanceList:instances, ErrorChan:respChan}
}

func (manager *InstanceManager) MigrateInstances(instances []string, respChan chan error){
	manager.commands <- instanceCommand{Type: InsCmdMigrateInstance, InstanceList:instances, ErrorChan:respChan}
}

func (manager *InstanceManager) ResetGuestSystem(id string, resp chan error){
	manager.commands <- instanceCommand{Type: InsCmdResetSystem, Instance: id, ErrorChan:resp}
}

func (manager *InstanceManager) ResetMonitorPassword(id string, respChan chan InstanceResult){
	manager.commands <- instanceCommand{Type: InsCmdResetMonitorSecret, Instance: id, ResultChan: respChan}
}


func (manager *InstanceManager) SyncAddressAllocation(allocationMode string){
	manager.commands <- instanceCommand{Type: InsCmdSyncAddressAllocation, Allocation: allocationMode}
}

type instanceDataConfig struct {
	Instances   []GuestConfig `json:"instances"`
	StoragePool string        `json:"storage_pool,omitempty"`
	StorageURL  string        `json:"storage_url,omitempty"`
}

func (manager *InstanceManager) saveInstanceConfig(instanceID string) (err error){
	if DefaultLocalPoolName != manager.storagePool{
		//share pool available
		var metaFile = filepath.Join(manager.storageURL, fmt.Sprintf("%s.%s", instanceID, MetaFileSuffix))
		ins, exists := manager.instances[instanceID]
		if !exists{
			err = fmt.Errorf("invalid instance '%s'", instanceID)
			return
		}
		data, err := json.MarshalIndent(ins.GuestConfig, "", " ")
		if err != nil {
			return err
		}
		if err = ioutil.WriteFile(metaFile, data, ConfigFilePerm); err != nil {
			return err
		}
		log.Printf("<instance> meta data of instance '%s' saved into '%s'", ins.Name, metaFile)
	}
	return manager.saveConfig()
}

func (manager *InstanceManager) removeInstanceConfig(instanceID string) (err error){
	if DefaultLocalPoolName != manager.storagePool{
		//share pool available
		var metaFile = filepath.Join(manager.storageURL, fmt.Sprintf("%s.%s", instanceID, MetaFileSuffix))
		if _, err = os.Stat(metaFile); !os.IsNotExist(err){
			//exists
			if err = os.Remove(metaFile);err != nil{
				log.Printf("<instance> warning: remove meta file %s fail: %s", metaFile, err.Error())
			}else{
				log.Printf("<instance> metafile '%s' removed", metaFile)
			}
		}else{
			log.Printf("<instance> warning: can not find meta file '%s'", metaFile)
		}
	}
	return manager.saveConfig()
}

func (manager *InstanceManager) saveConfig() error {
	var config instanceDataConfig
	for _, ins := range manager.instances {
		config.Instances = append(config.Instances, ins.GuestConfig)
	}
	config.StoragePool = manager.storagePool
	config.StorageURL = manager.storageURL
	data, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(manager.dataFile, data, ConfigFilePerm); err != nil {
		return err
	}
	log.Printf("<instance> %d guest(es) saved into '%s'", len(config.Instances), manager.dataFile)
	return nil
}

func (manager *InstanceManager) loadConfig() error {
	if _, err := os.Stat(manager.dataFile); os.IsNotExist(err) {
		log.Println("<instance> no instance data available")
		return nil
	}
	data, err := ioutil.ReadFile(manager.dataFile)
	if err != nil {
		return err
	}
	var config instanceDataConfig
	if err = json.Unmarshal(data, &config); err != nil {
		return err
	}
	for _, ins := range config.Instances {
		realStatus, err := manager.util.GetInstanceStatus(ins.ID)
		if err != nil {
			return err
		}
		ins.Running = realStatus.Running
		ins.Created = true
		if "" == ins.AuthUser{
			if SystemNameWindows == ins.Template.OperatingSystem{
				ins.AuthUser = AdminWindows
			}else{
				ins.AuthUser = AdminLinux
			}
		}
		if realStatus.Running {
			realStatus.GuestConfig = ins
			manager.StartCPUMonitor(&realStatus)
			manager.instances[ins.ID] = realStatus
		} else {
			//load as static config
			manager.instances[ins.ID] = InstanceStatus{GuestConfig: ins}
		}
	}
	if "" != config.StoragePool{
		manager.storagePool = config.StoragePool
		manager.storageURL = config.StorageURL
	}else{
		manager.storagePool = DefaultLocalPoolName
	}
	log.Printf("<instance> %d guest(es) loaded from '%s', current storage '%s'", len(manager.instances),
		manager.dataFile, manager.storagePool)
	return nil
}

func (manager *InstanceManager) handleCommand(cmd instanceCommand) {
	var err error
	switch cmd.Type {
	case InsCmdCreate:
		err = manager.handleCreateInstance(cmd.Config, cmd.ErrorChan)
	case InsCmdDelete:
		err = manager.handleDeleteInstance(cmd.Instance, cmd.ErrorChan)
	case InsCmdGet:
		err = manager.handleGetInstanceConfig(cmd.Instance, cmd.ResultChan)
	case InsCmdGetStatus:
		err = manager.handleGetInstanceStatus(cmd.Instance, cmd.ResultChan)
	case InsCmdGetAllConfig:
		err = manager.handleGetAllConfig(cmd.AllConfigChan)
	case InsCmdStart:
		if cmd.Media.ID != "" {
			err = manager.handleStartInstanceWithMedia(cmd.Instance, cmd.Media, cmd.ErrorChan)
		} else {
			err = manager.handleStartInstance(cmd.Instance, cmd.ErrorChan)
		}
	case InsCmdStop:
		err = manager.handleStopInstance(cmd.Instance, cmd.Reboot, cmd.Force, cmd.ErrorChan)
	case InsCmdIsRunning:
		err = manager.handleIsInstanceRunning(cmd.Instance, cmd.BoolChan)
	case InsCmdRename:
		err = manager.handleModifyGuestName(cmd.Instance, cmd.Name, cmd.ErrorChan)
	case InsCmdModifyCore:
		err = manager.handleModifyGuestCore(cmd.Instance, cmd.Cores, cmd.ErrorChan)
	case InsCmdModifyMemory:
		err = manager.handleModifyGuestMemory(cmd.Instance, cmd.Memory, cmd.ErrorChan)
	case InsCmdModifyCPUPriority:
		err = manager.handleModifyCPUPriority(cmd.Instance, cmd.Priority, cmd.ErrorChan)
	case InsCmdModifyDiskThreshold:
		err = manager.handleModifyDiskThreshold(cmd.Instance, cmd.ReadSpeed, cmd.ReadIOPS, cmd.WriteSpeed, cmd.WriteIOPS, cmd.ErrorChan)
	case InsCmdModifyNetworkThreshold:
		err = manager.handleModifyNetworkThreshold(cmd.Instance, cmd.ReceiveSpeed, cmd.SendSpeed, cmd.ErrorChan)
	case InsCmdModifyAuth:
		err = manager.handleModifyGuestAuth(cmd.Instance, cmd.Password, cmd.User, cmd.ResultChan)
	case InsCmdGetAuth:
		err = manager.handleGetGuestAuth(cmd.Instance, cmd.ResultChan)
	case InsCmdFinishInitialize:
		err = manager.handleFinishGuestInitialize(cmd.Instance, cmd.ErrorChan)
	case InsCmdResetSystem:
		err = manager.handleResetGuestSystem(cmd.Instance, cmd.ErrorChan)
	case InsCmdUpdateDiskSize:
		err = manager.handleUpdateDiskSize(cmd.Instance, cmd.Index, cmd.Size, cmd.ErrorChan)
	case InsCmdAddEventListener:
		err = manager.handleAddEventListener(cmd.Name, cmd.EventChan)
	case InsCmdRemoveEventListener:
		err = manager.handleRemoveEventListener(cmd.Name)
	case InsCmdInsertMedia:
		err = manager.handleAttachMedia(cmd.Instance, cmd.Media, cmd.ErrorChan)
	case InsCmdEjectMedia:
		err = manager.handleDetachMedia(cmd.Instance, cmd.ErrorChan)
	case InsCmdUsingStorage:
		err = manager.handleUsingStorage(cmd.Name, cmd.URL, cmd.ErrorChan)
	case InsCmdDetachStorage:
		err = manager.handleDetachStorage(cmd.ErrorChan)
	case InsCmdGetNetworkResource:
		err = manager.handleGetNetworkResources(cmd.InstanceList, cmd.ResultChan)
	case InsCmdAttachInstance:
		err = manager.handleAttachInstances(cmd.NetworkResource, cmd.ErrorChan)
	case InsCmdDetachInstance:
		err = manager.handleDetachInstances(cmd.InstanceList, cmd.ErrorChan)
	case InsCmdMigrateInstance:
		err = manager.handleMigrateInstances(cmd.InstanceList, cmd.ErrorChan)
	case InsCmdResetMonitorSecret:
		err = manager.handleResetMonitorPassword(cmd.Instance, cmd.ResultChan)
	case InsCmdSyncAddressAllocation:
		err = manager.handleSyncAddressAllocation(cmd.Allocation)
	default:
		log.Printf("<instance> unsupported command type %d", cmd.Type)
	}
	if err != nil {
		log.Printf("<instance> handle command %s fail: %s", cmd.Type.toString(), err.Error())
	}
}

func (manager *InstanceManager) handleCreateInstance(config GuestConfig, resp chan error) error {
	if config.ID == "" {
		err := errors.New("instance id required")
		resp <- err
		return err
	}
	if _, exists := manager.instances[config.ID]; exists {
		err := fmt.Errorf("instance '%s' already exist", config.ID)
		resp <- err
		return err
	}
	if nil == config.Template{
		config.Template = &manager.defaultTemplate
		log.Printf("<instance> using default template for instance '%s'", config.Name)
	}
	guest, err := manager.util.CreateInstance(config)
	if err != nil {
		resp <- err
		return err
	}
	guest.CreateTime = time.Now().Format(TimeFormatLayout)
	manager.instances[guest.ID] = InstanceStatus{GuestConfig: guest}
	log.Printf("<instance> new instance '%s'(id '%s') created", guest.Name, guest.ID)
	resp <- nil
	return manager.saveInstanceConfig(config.ID)
}

func (manager *InstanceManager) handleDeleteInstance(id string, resp chan error) error {
	err := manager.util.DeleteInstance(id)
	if err != nil {
		resp <- err
		return err
	}
	delete(manager.instances, id)
	log.Printf("<instance> instance '%s' deleted", id)
	resp <- nil
	return manager.removeInstanceConfig(id)
}

func (manager *InstanceManager) handleGetInstanceConfig(id string, resp chan InstanceResult) error {
	ins, exist := manager.instances[id]
	if !exist {
		err := fmt.Errorf("invalid instance '%s'", id)
		resp <- InstanceResult{Error: err}
		return err
	}
	resp <- InstanceResult{Instance: ins}
	return nil
}

func (manager *InstanceManager) handleGetInstanceStatus(id string, resp chan InstanceResult) error {
	ins, exist := manager.instances[id]
	if !exist {
		err := fmt.Errorf("invalid instance '%s'", id)
		resp <- InstanceResult{Error: err}
		return err
	}
	if !ins.Created {
		//not created
		resp <- InstanceResult{Instance: ins}
		return nil
	}
	status, err := manager.util.GetInstanceStatus(id)
	if err != nil {
		resp <- InstanceResult{Error: err}
		return err
	}
	if ins.Running != status.Running {
		if status.Running {
			//stopped => running

			log.Printf("<instance> detected instance started when get status of '%s'", ins.Name)
			manager.events <- InstanceStatusChangedEvent{ID: id, Event: InstanceStarted, Timestamp: time.Now()}
		} else {
			//running => stopped
			log.Printf("<instance> detected instance stopped when get status of '%s'", ins.Name)
			manager.events <- InstanceStatusChangedEvent{ID: id, Event: InstanceStopped, Timestamp: time.Now()}
		}
		ins.Running = status.Running
	}
	ins.AvailableMemory = status.AvailableMemory
	ins.AvailableDisk = status.AvailableDisk
	ins.BytesRead = status.BytesRead
	ins.BytesWritten = status.BytesWritten
	ins.BytesReceived = status.BytesReceived
	ins.BytesSent = status.BytesSent
	//update
	manager.instances[id] = ins
	resp <- InstanceResult{Instance: ins}
	return nil
}

func (manager *InstanceManager) handleIsInstanceRunning(id string, respChan chan bool) error {
	running, err := manager.util.IsInstanceRunning(id)
	if err != nil {
		respChan <- false
		return err
	}
	respChan <- running
	return nil
}

func (manager *InstanceManager) handleStartInstance(id string, resp chan error) error {
	ins, exist := manager.instances[id]
	if !exist {
		err := fmt.Errorf("invalid instance '%s'", id)
		resp <- err
		return err
	}
	if ins.Running {
		err := fmt.Errorf("instance '%s' already started", id)
		resp <- err
		return err
	}
	if err := manager.util.StartInstance(id); err != nil {
		resp <- err
		return err
	}
	log.Printf("<instance> instance '%s' started", id)
	ins.Running = true
	manager.StartCPUMonitor(&ins)
	manager.instances[id] = ins
	resp <- nil
	return nil
}
func (manager *InstanceManager) handleStartInstanceWithMedia(id string, media InstanceMediaConfig, resp chan error) error {
	ins, exist := manager.instances[id]
	if !exist {
		err := fmt.Errorf("invalid instance '%s'", id)
		resp <- err
		return err
	}
	if ins.Running {
		err := fmt.Errorf("instance '%s' already started", id)
		resp <- err
		return err
	}
	var resourceURI = manager.apiPath(fmt.Sprintf("/%s/%s/file/", MediaImagePath, media.ID))
	if err := manager.util.StartInstanceWithMedia(id, media.Host, resourceURI, media.Port); err != nil {
		resp <- err
		return err
	}

	log.Printf("<instance> instance '%s' started with media '%s'", id, media.ID)

	ins.Running = true
	ins.MediaAttached = true
	ins.MediaSource = media.ID
	manager.StartCPUMonitor(&ins)
	manager.instances[id] = ins
	resp <- nil
	return nil
}

func (manager *InstanceManager) handleStopInstance(id string, reboot, force bool, resp chan error) error {
	ins, exist := manager.instances[id]
	if !exist {
		err := fmt.Errorf("invalid instance '%s'", id)
		resp <- err
		return err
	}
	if !ins.Running {
		err := fmt.Errorf("instance '%s' alreay stopped", id)
		resp <- err
		return err
	}
	if err := manager.util.StopInstance(id, reboot, force); err != nil {
		resp <- err
		return err
	}
	if !reboot {
		running, err := manager.util.IsInstanceRunning(id)
		if err != nil {
			return err
		}
		if !running {
			log.Printf("<instance> instance '%s' stopped", id)
			ins.Running = false
			ins.MediaAttached = false
			ins.MediaSource = ""
			ins.CpuUsages = 0.0
			manager.instances[id] = ins
		} else {
			log.Printf("<instance> warning: instance '%s' still running", id)
		}

	} else {
		log.Printf("<instance> instance '%s' reboot", id)
	}
	resp <- nil
	return nil
}

func (manager *InstanceManager) handleGetAllConfig(respChan chan []GuestConfig) error {
	var allConfig []GuestConfig
	for _, ins := range manager.instances {
		allConfig = append(allConfig, ins.GuestConfig)
	}
	respChan <- allConfig
	return nil
}

func (manager *InstanceManager) handleModifyGuestName(id, name string, resp chan error) (err error){
	ins, exists := manager.instances[id]
	if !exists{
		err = fmt.Errorf("invalid guest '%s'", id)
		resp <- err
		return err
	}
	if err = manager.util.Rename(id, name); err != nil{
		resp <- err
		return err
	}
	log.Printf("<instance> guest '%s' renamed to '%s'", ins.Name, name)
	ins.Name = name
	manager.instances[id] = ins
	resp <- nil
	return nil
}

func (manager *InstanceManager) handleModifyGuestCore(id string, core uint, resp chan error) error {
	current, exists := manager.instances[id]
	var err error
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", id)
		resp <- err
		return err
	}
	if current.Cores == core {
		err = errors.New("no need to change")
		resp <- err
		return err
	}

	if core > current.Cores {
		//update topology
		if err = manager.util.ModifyCPUTopology(id, core, false); err != nil {
			resp <- err
			return err
		}
	}
	if err = manager.util.ModifyCore(id, core, false); err != nil {
		resp <- err
		return err
	}
	log.Printf("<instance> cores of instance '%s' changed to %d when next start", current.Name, core)
	current.Cores = core
	manager.instances[id] = current
	resp <- nil
	return manager.saveInstanceConfig(id)
}

func (manager *InstanceManager) handleModifyGuestMemory(id string, memory uint, resp chan error) error {
	current, exists := manager.instances[id]
	var err error
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", id)
		resp <- err
		return err
	}
	if current.Memory == memory {
		err = errors.New("no need to change")
		resp <- err
		return err
	}
	if err = manager.util.ModifyMemory(id, memory, false); err != nil {
		resp <- err
		return err
	}
	log.Printf("<instance> memory of instance '%s' changed to %d MiB when next start", current.Name, memory>>10)
	current.Memory = memory
	manager.instances[id] = current
	resp <- nil
	return manager.saveInstanceConfig(id)
}

func (manager *InstanceManager) handleModifyCPUPriority(guestID string, priority PriorityEnum, resp chan error) (err error){
	currentGuest, exists := manager.instances[guestID]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", guestID)
		resp <- err
		return err
	}
	if currentGuest.CPUPriority == priority{
		err = errors.New("no need to change")
		resp <- err
		return err
	}
	if err = manager.util.SetCPUThreshold(guestID, priority); err != nil{
		resp <- err
		return err
	}
	log.Printf("<instance> CPU priority of guest '%s' changed to %d", guestID, priority)
	currentGuest.CPUPriority = priority
	manager.instances[guestID] = currentGuest
	resp <- nil
	return  manager.saveInstanceConfig(guestID)
}

func (manager *InstanceManager) handleModifyDiskThreshold(guestID string, readSpeed, readIOPS, writeSpeed, writeIOPS uint64, resp chan error) (err error){
	currentGuest, exists := manager.instances[guestID]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", guestID)
		resp <- err
		return err
	}
	if currentGuest.ReadSpeed == readSpeed && currentGuest.ReadIOPS == readIOPS && currentGuest.WriteSpeed == writeSpeed && currentGuest.WriteIOPS == writeIOPS{
		err = errors.New("no need to change")
		resp <- err
		return err
	}

	if err = manager.util.SetDiskThreshold(guestID, readSpeed, readIOPS, writeSpeed, writeIOPS); err != nil{
		resp <- err
		return err
	}
	//log.Printf("<instance> disk limit of guest '%s' changed to read %d MB / %d, write %d MB / %d", guestID,
	//	readSpeed >> 20, readIOPS, writeSpeed >> 20, writeIOPS)
	log.Printf("<instance> disk IO limit of guest '%s' changed to read %d, write %d per second", guestID,
		readIOPS, writeIOPS)
	currentGuest.ReadSpeed = readSpeed
	currentGuest.ReadIOPS = readIOPS
	currentGuest.WriteSpeed = writeSpeed
	currentGuest.WriteIOPS = writeIOPS
	manager.instances[guestID] = currentGuest
	resp <- nil
	return  manager.saveInstanceConfig(guestID)
}

func (manager *InstanceManager) handleModifyNetworkThreshold(guestID string, receive, send uint64, resp chan error) (err error){
	instance, exists := manager.instances[guestID]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", guestID)
		resp <- err
		return err
	}
	if instance.ReceiveSpeed == receive && instance.SendSpeed == send{
		err = errors.New("no need to change")
		resp <- err
		return err
	}

	if err = manager.util.SetNetworkThreshold(guestID, receive, send); err != nil{
		resp <- err
		return err
	}
	log.Printf("<instance> network limit of guest '%s' changed to receive %d Kps / send %d Kps", guestID,
		receive >> 10, send >> 10)
	instance.ReceiveSpeed = receive
	instance.SendSpeed = send
	manager.instances[guestID] = instance
	resp <- nil
	return  manager.saveInstanceConfig(guestID)
}

func (manager *InstanceManager) handleModifyGuestAuth(id, password, user string, resp chan InstanceResult) (err error) {
	ins, exists := manager.instances[id]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", id)
		resp <- InstanceResult{Error: err}
		return err
	}
	if "" == user {
		user = ins.AuthUser
	}
	err = manager.util.ModifyPassword(id, user, password)
	if err != nil {
		resp <- InstanceResult{Error: err}
		return err
	}
	//update
	ins.AuthUser = user
	ins.AuthSecret = password
	manager.instances[id] = ins
	resp <- InstanceResult{User: user, Password: password}
	return manager.saveInstanceConfig(id)
}
func (manager *InstanceManager) handleGetGuestAuth(id string, resp chan InstanceResult) (err error) {
	ins, exists := manager.instances[id]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", id)
		resp <- InstanceResult{Error: err}
		return err
	}
	resp <- InstanceResult{User: ins.AuthUser, Password: ins.AuthSecret}
	return nil
}
func (manager *InstanceManager) handleFinishGuestInitialize(id string, resp chan error) (err error){
	ins, exists := manager.instances[id]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", id)
		resp <- err
		return err
	}
	if ins.Initialized{
		err = fmt.Errorf("guest '%s' already initialized", id)
		resp <- err
		return err
	}
	ins.Initialized = true
	manager.instances[id] = ins
	log.Printf("<instance> guest '%s' initialized", id)
	resp <- nil
	return manager.saveInstanceConfig(id)
}

func (manager *InstanceManager) handleResetGuestSystem(guestID string, resp chan error) (err error){
	ins, exists := manager.instances[guestID]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", guestID)
		resp <- err
		return err
	}
	if !ins.Initialized{
		log.Printf("<instance> warning: guest '%s' not initialized before reset", guestID)
	}else{
		ins.Initialized = false
	}
	manager.instances[guestID] = ins
	log.Printf("<instance> guest '%s' resetted", guestID)
	resp <- nil
	return manager.saveInstanceConfig(guestID)
}

func (manager *InstanceManager) handleUpdateDiskSize(guest string, index int, size uint64, resp chan error) (err error) {
	ins, exists := manager.instances[guest]
	if !exists {
		err = fmt.Errorf("invalid guest '%s'", guest)
		resp <- err
		return err
	}
	if index >= len(ins.Disks) {
		err = fmt.Errorf("invalid disk index %d of guest '%s'", index, guest)
		resp <- err
		return err
	}
	ins.Disks[index] = size
	manager.instances[guest] = ins
	resp <- nil
	return manager.saveInstanceConfig(guest)
}

func (manager *InstanceManager) handleAddEventListener(listener string, eventChan chan InstanceStatusChangedEvent) (err error){
	_, exists := manager.eventListeners[listener]
	if exists{
		err = fmt.Errorf("listener '%s' already exists", listener)
		return
	}
	manager.eventListeners[listener] = eventChan
	log.Printf("<instance> event listener '%s' added", listener)
	return nil
}

func (manager *InstanceManager) handleAttachMedia(id string, media InstanceMediaConfig, resp chan error) (err error){
	ins, exists := manager.instances[id]
	if !exists {
		err = fmt.Errorf("invalid instance '%s'", id)
		resp <- err
		return err
	}
	if ins.MediaAttached{
		err = fmt.Errorf("instance '%s' already has media attached", id)
		resp <- err
		return err
	}
	var uri = manager.apiPath(fmt.Sprintf("/%s/%s/file/", MediaImagePath, media.ID))
	if err = manager.util.InsertMedia(id, media.Host, uri, media.Port); err != nil{
		resp <- err
		return err
	}
	ins.MediaAttached = true
	ins.MediaSource = media.ID
	manager.instances[id] = ins
	resp <- nil
	log.Printf("<instance> media '%s' attached to '%s'", media.ID, ins.Name)
	return nil
}
func (manager *InstanceManager) handleDetachMedia(id string, resp chan error)(err error){
	ins, exists := manager.instances[id]
	if !exists {
		err = fmt.Errorf("invalid instance '%s'", id)
		resp <- err
		return err
	}
	if !ins.MediaAttached{
		err = fmt.Errorf("no media attached with instance '%s'", id)
		resp <- err
		return err
	}
	if err = manager.util.EjectMedia(id); err != nil{
		resp <- err
		return err
	}
	log.Printf("<instance> media '%s' detached from '%s'", ins.MediaSource, ins.Name)
	ins.MediaAttached = false
	ins.MediaSource = ""
	manager.instances[id] = ins
	resp <- nil
	return nil
}

func (manager *InstanceManager) handleUsingStorage(name, url string, respChan chan error) (err error){
	if name == manager.storagePool{
		log.Printf("<instance> no need to change storage '%s'", name)
		respChan <- nil
		return nil
	}
	manager.storagePool = name
	manager.storageURL = url
	if _, err = os.Stat(url); os.IsNotExist(err){
		err = fmt.Errorf("invalid image path '%s'", url)
		respChan <- err
		return err
	}
	log.Printf("<instance> using path '%s' as storage '%s'", url, name)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *InstanceManager) handleDetachStorage(respChan chan error) (err error){
	if DefaultLocalPoolName == manager.storagePool{
		log.Printf("<instance> no need to detach local storage '%s'", manager.storagePool)
		respChan <- nil
		return nil
	}
	log.Printf("<instance> detached from storage '%s'", manager.storagePool)
	manager.storagePool = DefaultLocalPoolName
	manager.storageURL = ""
	respChan <- nil
	return manager.saveConfig()
}


func (manager *InstanceManager) handleGetNetworkResources(instances []string, respChan chan InstanceResult) (err error){
	var result = map[string]InstanceNetworkResource{}
	var sharedStorageEnabled = DefaultLocalPoolName != manager.storagePool
	for _, instanceID := range instances{
		if status, exists := manager.instances[instanceID]; exists{
			result[instanceID] = InstanceNetworkResource{int(status.MonitorPort), status.HardwareAddress, status.InternalAddress, status.ExternalAddress}
		}else if !sharedStorageEnabled {
			err = fmt.Errorf("shared storage required for load meta data for instance '%s'", instanceID)
			respChan <- InstanceResult{Error:err}
			return 
		}else{
			var metaFile = filepath.Join(manager.storageURL, fmt.Sprintf("%s.%s", instanceID, MetaFileSuffix))
			if _, err = os.Stat(metaFile); os.IsNotExist(err){
				err = fmt.Errorf("meta file '%s' of instance '%s' not exists", metaFile, instanceID)
				respChan <- InstanceResult{Error:err}
				return
			}
			data, err := ioutil.ReadFile(metaFile)
			if err != nil{
				log.Printf("<instance> read meta file for instance '%s' fail: %s", instanceID, err.Error())
				respChan <- InstanceResult{Error:err}
				return err
			}
			var ins = InstanceStatus{}
			if err = json.Unmarshal(data, &ins.GuestConfig); err != nil{
				log.Printf("<instance> load config for instance '%s' fail: %s", instanceID, err.Error())
				respChan <- InstanceResult{Error:err}
				return err
			}
			result[instanceID] = InstanceNetworkResource{int(ins.MonitorPort), ins.HardwareAddress, ins.InternalAddress, ins.ExternalAddress}
		}
	}
	respChan <- InstanceResult{NetworkResources:result}
	return nil
}

func (manager *InstanceManager) handleAttachInstances(resources map[string]InstanceNetworkResource, respChan chan error) (err error){
	if DefaultLocalPoolName == manager.storagePool{
		err = errors.New("attach instance not support by the local storage")
		respChan <- err
		return err
	}
	for instanceID, resource := range resources{
		var metaFile = filepath.Join(manager.storageURL, fmt.Sprintf("%s.%s", instanceID, MetaFileSuffix))
		if _, err = os.Stat(metaFile); os.IsNotExist(err){
			err = fmt.Errorf("meta file '%s' of instance '%s' not exists", metaFile, instanceID)
			respChan <- err
			return err
		}
		data, err := ioutil.ReadFile(metaFile)
		if err != nil{
			log.Printf("<instance> read meta file for instance '%s' fail: %s", instanceID, err.Error())
			respChan <- err
			return err
		}
		var ins = InstanceStatus{}
		if err = json.Unmarshal(data, &ins.GuestConfig); err != nil{
			log.Printf("<instance> load config for instance '%s' fail: %s", instanceID, err.Error())
			respChan <- err
			return err
		}
		if nil == ins.Template{
			ins.Template = &manager.defaultTemplate
			log.Printf("<instance> using default template for instance '%s'", ins.Name)
		}
		ins.MonitorPort = uint(resource.MonitorPort)
		if ins.GuestConfig, err = manager.util.CreateInstance(ins.GuestConfig);err != nil{
			log.Printf("<instance> resume instance '%s' fail: %s", ins.Name, err.Error())
			respChan <- err
			return err
		}
		data, err = json.MarshalIndent(ins, "", " ")
		if err != nil{
			log.Printf("<instance> generate meta data for instance '%s' fail: %s", ins.Name, err.Error())
			respChan <- err
			return err
		}
		if err = ioutil.WriteFile(metaFile, data, ConfigFilePerm); err != nil{
			log.Printf("<instance> write meta file for instance '%s' fail: %s", ins.Name, err.Error())
			respChan <- err
			return err
		}
		manager.instances[instanceID] = ins
		log.Printf("<instance> instance '%s' attached with monitor port %d", ins.Name, ins.MonitorPort)
	}
	log.Printf("<instance> %d instance(s) attached", len(resources))
	respChan <- nil
	return manager.saveConfig()
}

func (manager *InstanceManager) handleDetachInstances(instances []string, respChan chan error) (err error){
	if DefaultLocalPoolName == manager.storagePool{
		err = errors.New("detach instance not support by the local storage")
		respChan <- err
		return err
	}
	if 0 == len(instances){
		for id, _ := range manager.instances{
			instances = append(instances, id)
		}
	}
	for _, instanceID := range instances {
		ins, exists := manager.instances[instanceID]
		if !exists {
			err = fmt.Errorf("invalid instance '%s'", instanceID)
			respChan <- err
			return err
		}
		if ins.Running{
			if err = manager.util.StopInstance(instanceID, false, true);err != nil{
				log.Printf("<instance> shutdown instance '%s' for detaching fail: %s", ins.Name, err.Error())
				respChan <- err
				return err
			}
			log.Printf("<instance> instance '%s' shutdown for detaching", ins.Name)
		}
		if err = manager.util.DeleteInstance(instanceID);err != nil{
			log.Printf("<instance> detach instance '%s'('%s') fail: %s", ins.Name, instanceID, err.Error())
			respChan <- err
			return err
		}
		log.Printf("<instance> instance '%s' detached", ins.Name)
		delete(manager.instances, instanceID)
	}
	log.Printf("<instance> %d instance(s) detached", len(instances))
	respChan <- nil
	return manager.saveConfig()
}

func (manager *InstanceManager) handleMigrateInstances(instances []string, respChan chan error) (err error){
	if DefaultLocalPoolName == manager.storagePool{
		err = errors.New("migrate instance not support by the local storage")
		respChan <- err
		return err
	}
	for _, instanceID := range instances{
		ins, exists := manager.instances[instanceID]
		if !exists{
			err = fmt.Errorf("invalid instance '%s'", instanceID)
			respChan <- err
			return err
		}
		//start autostart instance in share storage
		if ins.AutoStart && !ins.Running{
			if err = manager.util.StartInstance(instanceID);err != nil{
				log.Printf("<instance> start migrated instance '%s'('%s') fail: %s", ins.Name, instanceID, err.Error())
				respChan <- err
				return err
			}
			log.Printf("<instance> migrated instance '%s' start", ins.Name)
			//left for sync
		}
	}
	log.Printf("<instance> %d instance(s) migrated", len(instances))
	respChan <- nil
	return nil
}

func (manager *InstanceManager) handleRemoveEventListener(listener string) (err error) {
	_, exists := manager.eventListeners[listener]
	if !exists{
		err =fmt.Errorf("invalid listener '%s'", listener)
		return
	}
	delete(manager.eventListeners, listener)
	log.Printf("<instance> event listener '%s' removed", listener)
	return nil
}

func (manager *InstanceManager) handleResetMonitorPassword(instanceID string, respChan chan InstanceResult) (err error){
	defer func() {
		if err != nil{
			respChan <- InstanceResult{Error: err}
		}
	}()
	const (
		MonitorSecretLength = 12
	)
	var ins InstanceStatus
	var exists bool
	if ins, exists = manager.instances[instanceID]; !exists{
		err = fmt.Errorf("invalid instance '%s'", instanceID)
		return
	}
	var newSecret = manager.generatePassword(MonitorSecretLength)
	var monitorProtocol string
	if nil != ins.Template{
		monitorProtocol = ins.Template.Control
	}else{
		monitorProtocol = RemoteControlVNC
	}
	if err = manager.util.ResetMonitorSecret(instanceID, monitorProtocol, ins.MonitorPort, newSecret); err != nil{
		err = fmt.Errorf("reset monitor secret fail: %s", err.Error())
		return
	}
	ins.MonitorSecret = newSecret
	manager.instances[instanceID] = ins
	respChan <- InstanceResult{Password: newSecret}
	log.Printf("<instance> monitor secret of instance '%s' reset", ins.Name)
	return manager.saveConfig()
}

func (manager *InstanceManager) handleSyncAddressAllocation(allocationMode string) (err error){
	if AddressAllocationNone == allocationMode{
		return
	}
	var modified = false
	var modifiedCount = 0
	for id, ins := range manager.instances{
		if allocationMode != ins.AddressAllocation{
			ins.AddressAllocation = allocationMode
			if !modified{
				modified = true
			}
			manager.instances[id] = ins
			modifiedCount++
		}
	}
	if modified{
		log.Printf("<instance> update address allocation of %d instance(s) to mode '%s'", modifiedCount, allocationMode)
		return manager.saveConfig()
	}
	return
}

func (config *GuestConfig) Marshal(message framework.Message) error {
	message.SetString(framework.ParamKeyName, config.Name)
	message.SetString(framework.ParamKeyInstance, config.ID)
	message.SetString(framework.ParamKeyUser, config.User)
	message.SetString(framework.ParamKeyGroup, config.Group)
	message.SetString(framework.ParamKeySystem, config.System)
	message.SetUInt(framework.ParamKeyCore, config.Cores)

	if config.AutoStart {
		message.SetUIntArray(framework.ParamKeyOption, []uint64{1})
	} else {
		message.SetUIntArray(framework.ParamKeyOption, []uint64{0})
	}
	if config.Created {
		message.SetBoolean(framework.ParamKeyEnable, true)
	} else {
		message.SetBoolean(framework.ParamKeyEnable, false)
		message.SetUInt(framework.ParamKeyProgress, config.Progress)
	}

	if config.Running {
		message.SetUInt(framework.ParamKeyStatus, InstanceStatusRunning)
	} else {
		message.SetUInt(framework.ParamKeyStatus, InstanceStatusStopped)
	}

	message.SetUInt(framework.ParamKeyMonitor, config.MonitorPort)
	message.SetString(framework.ParamKeySecret, config.MonitorSecret)
	message.SetUInt(framework.ParamKeyMemory, config.Memory)
	message.SetUIntArray(framework.ParamKeyDisk, config.Disks)
	message.SetString(framework.ParamKeyAddress, config.NetworkAddress)
	message.SetString(framework.ParamKeyCreate, config.CreateTime)
	message.SetString(framework.ParamKeyHardware, config.HardwareAddress)
	//QoS
	message.SetUInt(framework.ParamKeyPriority, uint(config.CPUPriority))
	message.SetUIntArray(framework.ParamKeyLimit, []uint64{config.ReadSpeed, config.WriteSpeed, config.ReadIOPS,
		config.WriteIOPS, config.ReceiveSpeed, config.SendSpeed})
	return nil
}

func (status *InstanceStatus) Marshal(message framework.Message) error {
	if err := status.GuestConfig.Marshal(message); err != nil {
		return err
	}
	message.SetBoolean(framework.ParamKeyMedia, status.MediaAttached)
	message.SetFloat(framework.ParamKeyUsage, status.CpuUsages)
	message.SetUIntArray(framework.ParamKeyAvailable, []uint64{status.AvailableMemory, status.AvailableDisk})
	message.SetUIntArray(framework.ParamKeyIO,
		[]uint64{status.BytesRead, status.BytesWritten, status.BytesReceived, status.BytesSent})
	return nil
}

func (manager *InstanceManager) StartCPUMonitor(status *InstanceStatus) (error) {
	times, _, err := manager.util.GetCPUTimes(status.ID)
	if err != nil {
		log.Printf("<instance> start monitor CPU times of '%s' fail: %s", status.ID, err.Error())
		return err
	}
	status.lastCPUTimes = times
	var now = time.Now()
	status.lastCPUCheck = now
	status.startTime = now
	return nil
}

func (manager *InstanceManager) apiPath(path string) string{
	return fmt.Sprintf("%s/v%d%s", APIRoot, APIVersion, path)
}

func (manager *InstanceManager) generatePassword(length int) string {
	const (
		Letters = "~!@#$%^&*()_[]-=+0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	)
	var result = make([]byte, length)
	var n = len(Letters)
	for i := 0; i < length; i++ {
		result[i] = Letters[manager.randomGenerator.Intn(n)]
	}
	return string(result)
}
