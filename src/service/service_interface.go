package service

import (
	"github.com/project-nano/framework"
	"time"
)

const (
	ConfigFilePerm        = 0600
	DefaultOperateTimeout = 5 * time.Second
	APIRoot               = "/api"
	APIVersion            = 1
)

type InstanceResult struct {
	Error            error
	Instance         InstanceStatus
	Password         string
	User             string
	Policy           SecurityPolicy
	NetworkResources map[string]InstanceNetworkResource
}

type InstanceMediaConfig struct {
	Mode InstanceMediaMode
	ID   string
	URI  string
	Host string
	Port uint
}

type InstanceMediaMode int

const (
	MediaModeHTTPS = InstanceMediaMode(iota)
)

type InstanceModule interface {
	UsingStorage(name, url string, respChan chan error)
	DetachStorage(respChan chan error)
	CreateInstance(require GuestConfig, resp chan error)
	DeleteInstance(id string, resp chan error)
	GetInstanceConfig(id string, resp chan InstanceResult)
	GetInstanceStatus(id string, resp chan InstanceResult)
	GetAllInstance(resp chan []GuestConfig)
	ModifyGuestName(id, name string, resp chan error)
	ModifyGuestCore(id string, core uint, resp chan error)
	ModifyGuestMemory(id string, core uint, resp chan error)
	ModifyCPUPriority(guestID string, priority PriorityEnum, resp chan error)
	ModifyDiskThreshold(guestID string, readSpeed, readIOPS, writeSpeed, writeIOPS uint64, resp chan error)
	ModifyNetworkThreshold(guestID string, receive, send uint64, resp chan error)
	ModifyAutoStart(guestID string, enable bool, respChan chan error)

	ModifyGuestAuth(id, password, usr string, resp chan InstanceResult)
	GetGuestAuth(id string, resp chan InstanceResult)
	FinishGuestInitialize(id string, resp chan error)
	ResetGuestSystem(id string, resp chan error)
	UpdateDiskSize(guest string, index int, size uint64, resp chan error)

	StartInstance(id string, resp chan error)
	StartInstanceWithMedia(id string, media InstanceMediaConfig, resp chan error)
	//StartWithNetwork(id, network string, resp chan error)
	StopInstance(id string, reboot, force bool, resp chan error)
	AttachMedia(id string, media InstanceMediaConfig, resp chan error)
	DetachMedia(id string, resp chan error)
	//AttachDisk(id, target string, resp chan error)
	//DetachDisk(id, target string, resp chan error)
	IsInstanceRunning(id string, resp chan bool)
	GetNetworkResources(instances []string, respChan chan InstanceResult)
	AttachInstances(resources map[string]InstanceNetworkResource, respChan chan error)
	DetachInstances(instances []string, respChan chan error)
	MigrateInstances(instances []string, respChan chan error)
	ResetMonitorPassword(id string, respChan chan InstanceResult)
	SyncAddressAllocation(allocationMode string)
	//Security Policy
	GetSecurityPolicy(instanceID string, respChan chan InstanceResult)
	AddSecurityPolicyRule(instanceID string, rule SecurityPolicyRule, respChan chan error)
	ModifySecurityPolicyRule(instanceID string, index int, rule SecurityPolicyRule, respChan chan error)
	RemoveSecurityPolicyRule(instanceID string, index int, respChan chan error)
	ChangeDefaultSecurityPolicyAction(instanceID string, accept bool, respChan chan error)
	PullUpSecurityPolicyRule(instanceID string, index int, respChan chan error)
	PushDownSecurityPolicyRule(instanceID string, index int, respChan chan error)
}

type SnapshotConfig struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreateTime  string `json:"create_time"`
	IsRoot      bool   `json:"is_root"`
	IsCurrent   bool   `json:"is_current"`
	Backing     string `json:"backing,omitempty"`
	Running     bool   `json:"running"`
}

type AttachDeviceInfo struct {
	Name     string
	Protocol string
	Path     string
	Attached bool
	Error    string
}

type StorageResult struct {
	Error        error
	Volumes      []string
	Pool         string
	Image        string
	Size         uint
	Path         string
	Snapshot     SnapshotConfig
	SnapshotList []SnapshotConfig
	Devices      []AttachDeviceInfo
	StorageMode  StoragePoolMode
	SystemPaths  []string
	DataPaths    []string
}

type BootType int

const (
	BootTypeNone = BootType(iota)
	BootTypeCloudInit
)

type StorageModule interface {
	UsingStorage(name, protocol, host, target string, respChan chan StorageResult)
	DetachStorage(respChan chan error)
	GetAttachDevices(respChan chan StorageResult)
	CreateVolumes(groupName string, systemSize uint64, dataSize []uint64, bootType BootType, resp chan StorageResult)
	DeleteVolumes(groupName string, resp chan error)
	ReadDiskImage(id framework.SessionID, groupName, targetVol, sourceImage string, targetSize, imageSize uint64,
		mediaHost string, mediaPort uint, startChan chan error, progress chan uint, resultChan chan StorageResult)
	WriteDiskImage(id framework.SessionID, groupName, targetVol, sourceImage, mediaHost string, mediaPort uint,
		startChan chan error, progress chan uint, resultChan chan StorageResult)
	ResizeVolume(id framework.SessionID, groupName, targetVol string, targetSize uint64, respChan chan StorageResult)
	ShrinkVolume(id framework.SessionID, groupName, targetVol string, respChan chan StorageResult)
	//snapshot
	QuerySnapshot(groupName string, respChan chan StorageResult)
	GetSnapshot(groupName, snapshot string, respChan chan StorageResult)
	CreateSnapshot(groupName, snapshot, description string, respChan chan error)
	DeleteSnapshot(groupName, snapshot string, respChan chan error)
	RestoreSnapshot(groupName, snapshot string, respChan chan error)
	AttachVolumeGroup(groups []string, respChan chan error)
	DetachVolumeGroup(groups []string, respChan chan error)
	QueryStoragePaths(respChan chan StorageResult)
	ChangeDefaultStoragePath(target string, respChan chan error)
	ValidateVolumesForStart(groupName string, respChan chan error)
}

type NetworkResult struct {
	Error       error
	Name        string
	MonitorPort int
	External    string
	Internal    string
	Gateway     string
	DNS         []string
	Allocation  string
	Resources   map[string]InstanceNetworkResource
}

type NetworkModule interface {
	GetBridgeName() string
	GetCurrentConfig(resp chan NetworkResult)
	AllocateInstanceResource(instance, hwaddress, internal, external string, resp chan NetworkResult)
	DeallocateAllResource(instance string, resp chan error)
	AttachInstances(resources map[string]InstanceNetworkResource, resp chan NetworkResult)
	DetachInstances(instances []string, resp chan error)
	UpdateAddressAllocation(gateway string, dns []string, allocationMode string, resp chan error)
	GetAddressByHWAddress(hwaddress string, resp chan NetworkResult)
}
