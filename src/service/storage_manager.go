package service

import (
	"github.com/project-nano/framework"
	"github.com/libvirt/libvirt-go"
	"log"
	"path/filepath"
	"os"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"time"
	"os/exec"
	"github.com/pkg/errors"
)

type StorageVolume struct {
	Name       string `json:"name"`
	Format     string `json:"format"`
	Key        string `json:"key,omitempty"`
	Path       string `json:"path,omitempty"`
	Capacity   uint64 `json:"capacity,omitempty"`
	Allocation uint64 `json:"-"`
}

type StoragePool struct {
	Name         string          `json:"name"`
	ID           string          `json:"id,omitempty"`
	Mode         StoragePoolMode `json:"mode"`
	Target       string          `json:"target,omitempty"`
	Capacity     uint64          `json:"-"`
	Allocation   uint64          `json:"-"`
	Available    uint64          `json:"-"`
	SourceHost   string          `json:"source_host,omitempty"`
	SourceTarget string          `json:"source_target,omitempty"`
	AuthName     string          `json:"auth_name,omitempty"`
	AuthSecret   string          `json:"auth_secret,omitempty"`
}

type StoragePoolMode uint

const (
	StoragePoolModeLocal = iota
	StoragePoolModeNFS
)

type storageCommandType int

const (
	storageCommandCreateVolume    = iota
	storageCommandDeleteVolume
	storageCommandWriteDiskImage
	storageCommandReadDiskImage
	storageCommandResizeVolume
	storageCommandShrinkVolume
	storageCommandQuerySnapshot
	storageCommandGetSnapshot
	storageCommandCreateSnapshot
	storageCommandDeleteSnapshot
	storageCommandRestoreSnapshot
	storageCommandUsing
	storageCommandGetAttachDevice
	storageCommandDetachStorage
	storageCommandAttachVolume
	storageCommandDetachVolume
)

type storageCommand struct {
	Type          storageCommandType
	Instance      string
	SystemSize    uint64
	DataSize      []uint64
	Session       framework.SessionID
	Volume        string
	Image         string
	ImageSize     uint64
	Host          string
	Port          uint
	VolumeSize    uint64
	Snapshot      string
	Description   string
	BootImageType BootType
	Target        string
	Protocol      string
	Pool          string
	Groups        []string
	StartedChan   chan error
	ProgressChan  chan uint
	ResultChan    chan StorageResult
	ErrorChan     chan error
}

type InstanceVolume struct {
	StorageVolume
	Pool string `json:"pool"`
}

type InstanceVolumeGroup struct {
	System         InstanceVolume             `json:"system"`
	Data           []InstanceVolume           `json:"data"`
	BootImage      string                     `json:"boot_image,omitempty"`
	ActiveSnapshot string                     `json:"active_snapshot,omitempty"`
	BaseSnapshot   string                     `json:"base_snapshot,omitempty"`
	Snapshots      map[string]ManagedSnapshot `json:"snapshots,omitempty"`
	Locked         bool                       `json:"-"`
}

type ManagedStoragePool struct {
	StoragePool
	Volumes     map[string]bool `json:"volumes"`
	Attached    bool            `json:"-"`
	AttachError string          `json:"-"`
}

type ManagedSnapshot struct {
	SnapshotConfig
	Files map[string]string `json:"files"` //key = volume name, value = backing file path
}

type PendingTask struct {
	Progress     uint
	ProgressChan chan uint
	ResultChan   chan StorageResult
}

type StorageManager struct {
	commands         chan storageCommand
	dataFile         string
	dataPath         string
	localVolumesPath string
	currentPool      string
	nfsEnabled       bool
	pools            map[string]ManagedStoragePool
	schedulers       map[string]*IOScheduler //key = pool name
	groups           map[string]InstanceVolumeGroup
	tasks            map[framework.SessionID]PendingTask
	progressChan     chan SchedulerUpdate
	scheduleChan     chan SchedulerResult
	eventChan        chan schedulerEvent
	utility          *StorageUtility
	initiatorIP      string
	runner           *framework.SimpleRunner
}

const (
	DefaultLocalPoolName   = "local0"
	StoragePathPerm        = 0740
	DefaultLocalVolumePath = "/var/lib/libvirt/images"
	VolumeMetaFileSuffix   = "vols"
	FormatQcow2Suffix      = "qcow2"
)

const (
	StorageProtocolNFS = "nfs"
)

func CreateStorageManager(dataPath string, connect *libvirt.Connect) (*StorageManager, error) {
	const (
		StorageFilename       = "storage.data"
		DefaultQueueSize      = 1 << 10
	)
	var manager = StorageManager{}
	manager.commands = make(chan storageCommand, DefaultQueueSize)
	manager.progressChan = make(chan SchedulerUpdate, DefaultQueueSize)
	manager.scheduleChan = make(chan SchedulerResult, DefaultQueueSize)
	manager.eventChan = make(chan schedulerEvent, DefaultQueueSize)
	manager.dataFile = filepath.Join(dataPath, StorageFilename)
	manager.dataPath = dataPath
	manager.localVolumesPath = DefaultLocalVolumePath

	var err error
	manager.initiatorIP, err = GetCurrentIPOfDefaultBridge()
	if err != nil {
		log.Printf("<storage> get initiator ip fail: %s", err.Error())
		return nil, err
	}
	manager.utility, err = CreateStorageUtility(connect)
	if err != nil {
		return nil, err
	}
	manager.runner = framework.CreateSimpleRunner(manager.Routine)
	manager.pools = map[string]ManagedStoragePool{}
	manager.schedulers = map[string]*IOScheduler{}
	manager.groups = map[string]InstanceVolumeGroup{}
	manager.tasks = map[framework.SessionID]PendingTask{}
	if err = manager.loadConfig(); err != nil {
		return nil, err
	}
	return &manager, nil
}

func (manager *StorageManager) Start() error{
	return manager.runner.Start()
}

func (manager *StorageManager) Stop() error{
	return manager.runner.Stop()
}

func (manager *StorageManager) Routine(c framework.RoutineController) {
	log.Println("<storage> started")
	for poolName, scheduler := range manager.schedulers {
		if err := scheduler.Start(); err != nil {
			log.Printf("<storage> start scheduler for pool '%s' fail: %s", poolName, err.Error())
			return
		}
	}

	const (
		NotifyInterval = 2 * time.Second
	)
	var notifyTicker = time.NewTicker(NotifyInterval)

	for !c.IsStopping() {
		select {
		case <-c.GetNotifyChannel():
			c.SetStopping()
			break
		case cmd := <-manager.commands:
			manager.handleCommand(cmd)
		case <-notifyTicker.C:
			manager.notifyAllTask()
		case update := <-manager.progressChan:
			manager.handleSchedulerUpdate(update)
		case result := <-manager.scheduleChan:
			manager.handleSchedulerResult(result)
		case event := <-manager.eventChan:
			manager.handleSchedulerEvent(event)
		}
	}
	for poolName, scheduler := range manager.schedulers {
		if err := scheduler.Stop(); err != nil {
			log.Printf("<storage> warning:stop scheduler for pool '%s' fail: %s", poolName, err.Error())
		}
	}

	c.NotifyExit()
	log.Println("<storage> stopped")
}

func (manager *StorageManager) handleCommand(cmd storageCommand) {
	var err error
	switch cmd.Type {
	case storageCommandCreateVolume:
		err = manager.handleCreateVolumes(cmd.Instance, cmd.SystemSize, cmd.DataSize, cmd.BootImageType, cmd.ResultChan)
	case storageCommandDeleteVolume:
		err = manager.handleDeleteVolumes(cmd.Instance, cmd.ErrorChan)
	case storageCommandReadDiskImage:
		err = manager.handleReadDiskImage(cmd.Session, cmd.Instance, cmd.Volume, cmd.Image, cmd.SystemSize, cmd.ImageSize, cmd.Host, cmd.Port,
			cmd.StartedChan, cmd.ProgressChan, cmd.ResultChan)
	case storageCommandWriteDiskImage:
		err = manager.handleWriteDiskImage(cmd.Session, cmd.Instance, cmd.Volume, cmd.Image, cmd.Host, cmd.Port,
			cmd.StartedChan, cmd.ProgressChan, cmd.ResultChan)
	case storageCommandResizeVolume:
		err = manager.handleResizeVolume(cmd.Session, cmd.Instance, cmd.Volume, cmd.VolumeSize, cmd.ResultChan)
	case storageCommandShrinkVolume:
		err = manager.handleShrinkVolume(cmd.Session, cmd.Instance, cmd.Volume, cmd.ResultChan)
	case storageCommandQuerySnapshot:
		err = manager.handleQuerySnapshot(cmd.Instance, cmd.ResultChan)
	case storageCommandGetSnapshot:
		err = manager.handleGetSnapshot(cmd.Instance, cmd.Snapshot, cmd.ResultChan)
	case storageCommandCreateSnapshot:
		err = manager.handleCreateSnapshot(cmd.Instance, cmd.Snapshot, cmd.Description, cmd.ErrorChan)
	case storageCommandDeleteSnapshot:
		err = manager.handleDeleteSnapshot(cmd.Instance, cmd.Snapshot, cmd.ErrorChan)
	case storageCommandRestoreSnapshot:
		err = manager.handleRestoreSnapshot(cmd.Instance, cmd.Snapshot, cmd.ErrorChan)
	case storageCommandUsing:
		err = manager.handleUsingStorage(cmd.Pool, cmd.Protocol, cmd.Host, cmd.Target, cmd.ResultChan)
	case storageCommandGetAttachDevice:
		err = manager.handleGetAttachDevices(cmd.ResultChan)
	case storageCommandDetachStorage:
		err = manager.handleDetachStorage(cmd.ErrorChan)
	case storageCommandAttachVolume:
		err = manager.handleAttachVolumeGroup(cmd.Groups, cmd.ErrorChan)
	case storageCommandDetachVolume:
		err = manager.handleDetachVolumeGroup(cmd.Groups, cmd.ErrorChan)
	default:
		log.Printf("<storage> unsupported command type %d", cmd.Type)
	}
	if err != nil {
		log.Printf("<storage> handle command %d fail: %s", cmd.Type, err.Error())
	}
}

func (manager *StorageManager) UsingStorage(name, protocol, host, target string, respChan chan StorageResult){
	manager.commands <- storageCommand{Type:storageCommandUsing, Pool:name, Protocol:protocol, Host:host, Target:target, ResultChan:respChan}
}

func (manager *StorageManager) DetachStorage(respChan chan error){
	manager.commands <- storageCommand{Type: storageCommandDetachStorage, ErrorChan:respChan}
}

func (manager *StorageManager) GetAttachDevices(respChan chan StorageResult){
	manager.commands <- storageCommand{Type:storageCommandGetAttachDevice, ResultChan:respChan}
}

func (manager *StorageManager) CreateVolumes(groupName string, systemSize uint64, dataSize []uint64, bootType BootType, resp chan StorageResult) {
	cmd := storageCommand{Type: storageCommandCreateVolume, Instance: groupName, SystemSize: systemSize, DataSize: dataSize, BootImageType: bootType, ResultChan: resp}
	manager.commands <- cmd
}

func (manager *StorageManager) DeleteVolumes(groupName string, resp chan error) {
	cmd := storageCommand{Type: storageCommandDeleteVolume, Instance: groupName, ErrorChan: resp}
	manager.commands <- cmd
}

func (manager *StorageManager) ReadDiskImage(id framework.SessionID, groupName, targetVol, sourceImage string, targetSize, imageSize uint64, mediaHost string, mediaPort uint,
	startChan chan error, progress chan uint, resultChan chan StorageResult) {
	cmd := storageCommand{Type: storageCommandReadDiskImage, Session: id, Instance: groupName, Volume: targetVol, Image: sourceImage, SystemSize: targetSize,
		ImageSize: imageSize, Host: mediaHost, Port: mediaPort, StartedChan: startChan, ProgressChan: progress, ResultChan: resultChan}
	manager.commands <- cmd
}

func (manager *StorageManager) WriteDiskImage(id framework.SessionID, groupName, targetVol, sourceImage, mediaHost string, mediaPort uint,
	startChan chan error, progress chan uint, resultChan chan StorageResult) {
	cmd := storageCommand{Type: storageCommandWriteDiskImage, Session: id, Instance: groupName, Volume: targetVol, Image: sourceImage,
		Host: mediaHost, Port: mediaPort, StartedChan: startChan, ProgressChan: progress, ResultChan: resultChan}
	manager.commands <- cmd
}

func (manager *StorageManager) ResizeVolume(id framework.SessionID, groupName, targetVol string, targetSize uint64, respChan chan StorageResult) {
	manager.commands <- storageCommand{Type: storageCommandResizeVolume, Session: id, Instance: groupName, Volume: targetVol, VolumeSize: targetSize, ResultChan: respChan}
}

func (manager *StorageManager) ShrinkVolume(id framework.SessionID, groupName, targetVol string, respChan chan StorageResult) {
	manager.commands <- storageCommand{Type: storageCommandShrinkVolume, Session: id, Instance: groupName, Volume: targetVol, ResultChan: respChan}
}

func (manager *StorageManager) QuerySnapshot(groupName string, respChan chan StorageResult) {
	manager.commands <- storageCommand{Type: storageCommandQuerySnapshot, Instance: groupName, ResultChan: respChan}
}

func (manager *StorageManager) GetSnapshot(groupName, snapshot string, respChan chan StorageResult) {
	manager.commands <- storageCommand{Type: storageCommandGetSnapshot, Instance: groupName, Snapshot: snapshot, ResultChan: respChan}
}

func (manager *StorageManager) CreateSnapshot(groupName, snapshot, description string, respChan chan error) {
	manager.commands <- storageCommand{Type: storageCommandCreateSnapshot, Instance: groupName, Snapshot: snapshot, Description: description, ErrorChan: respChan}
}

func (manager *StorageManager) DeleteSnapshot(groupName, snapshot string, respChan chan error) {
	manager.commands <- storageCommand{Type: storageCommandDeleteSnapshot, Instance: groupName, Snapshot: snapshot, ErrorChan: respChan}
}

func (manager *StorageManager) RestoreSnapshot(groupName, snapshot string, respChan chan error) {
	manager.commands <- storageCommand{Type: storageCommandRestoreSnapshot, Instance: groupName, Snapshot: snapshot, ErrorChan: respChan}
}

func (manager *StorageManager) AttachVolumeGroup(groups []string, respChan chan error){
	manager.commands <- storageCommand{Type:storageCommandAttachVolume, Groups:groups, ErrorChan:respChan}
}

func (manager *StorageManager) DetachVolumeGroup(groups []string, respChan chan error){
	manager.commands <- storageCommand{Type:storageCommandDetachVolume, Groups:groups, ErrorChan:respChan}
}

type storageDataConfig struct {
	Pools       map[string]ManagedStoragePool  `json:"pools"`
	Groups      map[string]InstanceVolumeGroup `json:"groups"`
	CurrentPool string                         `json:"current_pool,omitempty"`
}

func (manager *StorageManager) saveVolumesMeta(groupName string) (err error){
	if currentPool, exists := manager.pools[manager.currentPool]; exists{
		if currentPool.Mode == StorageModeNFS{
			//save volume meta
			var metaFile = filepath.Join(currentPool.Target, fmt.Sprintf("%s.%s", groupName, VolumeMetaFileSuffix))
			group, exists := manager.groups[groupName]
			if !exists{
				err = fmt.Errorf("invalid group '%s'", groupName)
				return err
			}
			data, err := json.MarshalIndent(group, "", " ")
			if err != nil{
				return err
			}
			if err = ioutil.WriteFile(metaFile, data, ConfigFilePerm); err != nil{
				return err
			}
			log.Printf("<storage> volume meta for '%s' saved to '%s'", groupName, metaFile)
		}
	}
	return manager.saveConfig()
}

func (manager *StorageManager) removeVolumesMeta(groupName string) (err error){
	if currentPool, exists := manager.pools[manager.currentPool]; exists{
		if currentPool.Mode == StorageModeNFS{
			//save volume meta
			var metaFile = filepath.Join(currentPool.Target, fmt.Sprintf("%s.%s", groupName, VolumeMetaFileSuffix))
			if _, err = os.Stat(metaFile); !os.IsNotExist(err){
				if err = os.Remove(metaFile); err != nil{
					return err
				}
				log.Printf("<storage> volume meta for '%s' removed", groupName)
			}
		}
	}
	return manager.saveConfig()
}

func (manager *StorageManager) saveConfig() error {
	var config = storageDataConfig{}
	config.Pools = manager.pools
	config.Groups = manager.groups
	config.CurrentPool = manager.currentPool
	data, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(manager.dataFile, data, ConfigFilePerm); err != nil {
		return err
	}
	log.Printf("<storage> %d pools, %d group saved into '%s'", len(manager.pools), len(manager.groups), manager.dataFile)
	return nil
}

func (manager *StorageManager) loadConfig() error {
	if _, err := os.Stat(manager.dataFile); !os.IsNotExist(err){
		//exists
		data, err := ioutil.ReadFile(manager.dataFile)
		if err != nil {
			return err
		}
		var config storageDataConfig
		if err = json.Unmarshal(data, &config); err != nil {
			return err
		}
		if 0 != len(config.Pools) {

			manager.pools = config.Pools
			manager.groups = config.Groups
			manager.currentPool = config.CurrentPool
			if "" == manager.currentPool{
				manager.currentPool = DefaultLocalPoolName
			}
			for poolName, pool := range manager.pools {
				if pool.Mode == StoragePoolModeNFS{
					mounted, err := manager.utility.IsNFSPoolMounted(poolName)
					if mounted {
						pool.Attached = true
						log.Printf("<storage> nfs pool '%s' mounted", poolName)
					}else if poolName != manager.currentPool{
						pool.Attached = false
						pool.AttachError = err.Error()
						log.Printf("<storage> warning: check nfs pool '%s' fail: %s", poolName, err.Error())
					}else{
						//primary mount fail, must resume before start
						log.Printf("<storage> warning: primary nfs pool '%s' not mounted, try restart pool...", poolName)
						if err = manager.utility.StartPool(poolName); err != nil{
							pool.Attached = false
							pool.AttachError = err.Error()
							log.Printf("<storage> warning: restart nfs pool '%s' fail: %s", poolName, err.Error())
							return err
						}else{
							pool.Attached = true
							log.Printf("<storage> restart nfs pool '%s' success", poolName)
						}
					}
					if !manager.nfsEnabled{
						manager.nfsEnabled = true
					}
				}
				scheduler, err := CreateScheduler(poolName, manager.progressChan, manager.scheduleChan, manager.eventChan)
				if err != nil {
					return err
				}
				manager.schedulers[poolName] = scheduler
				manager.pools[poolName] = pool
			}
			log.Printf("<storage> %d pools, %d groups loaded, using pool '%s'", len(config.Pools), len(config.Groups), manager.currentPool)
			return nil
		}
	}
	log.Printf("<storage> no configure available in '%s'", manager.dataFile)
	if err := manager.generateDefaultPool(); err != nil {
		return err
	}
	return manager.saveConfig()
}

func (manager *StorageManager) generateDefaultPool() (err error) {
	var poolName = DefaultLocalPoolName
	var backingPool StoragePool
	if manager.utility.HasPool(poolName) {
		backingPool, err = manager.utility.GetPool(poolName)
		if err != nil {
			return err
		}
		log.Printf("<storage> found default storage pool '%s', path '%s'", poolName, backingPool.Target)
	} else {
		poolPath := filepath.Join(manager.localVolumesPath, poolName)
		if backingPool, err = manager.utility.CreateLocalPool(poolName, poolPath); err != nil {
			return err
		}
		log.Printf("<storage> new storage pool '%s' created, path '%s'", poolName, poolPath)
	}
	var defaultPool = ManagedStoragePool{StoragePool: backingPool, Volumes: map[string]bool{}}
	manager.pools[defaultPool.Name] = defaultPool
	manager.currentPool = poolName
	scheduler, err := CreateScheduler(defaultPool.Name, manager.progressChan, manager.scheduleChan, manager.eventChan)
	if err != nil {
		return err
	}
	manager.schedulers[defaultPool.Name] = scheduler
	return nil
}

func (manager *StorageManager) handleCreateVolumes(instanceID string, systemSize uint64, dataSize []uint64, bootType BootType, resp chan StorageResult) error {

	if _, exists := manager.groups[instanceID]; exists {
		err := fmt.Errorf("group %s already exists", instanceID)
		resp <- StorageResult{Error: err}
		return err
	}
	//using current pool
	var poolName = manager.currentPool
	pool, exists := manager.pools[poolName]
	if !exists {
		err := fmt.Errorf("current storage pool %s not exists", poolName)
		resp <- StorageResult{Error: err}
		return err
	}

	var systemDisk = fmt.Sprintf("%s_sys.%s", instanceID, FormatQcow2Suffix)
	var volNames = []string{systemDisk}
	var volSizes = []uint64{systemSize}
	if nil != dataSize {
		dataCount := len(dataSize)
		for i := 0; i < dataCount; i++ {
			name := fmt.Sprintf("%s_%d.%s", instanceID, i, FormatQcow2Suffix)
			volNames = append(volNames, name)
			volSizes = append(volSizes, dataSize[i])
		}
	}
	volumes, err := manager.utility.CreateVolumes(poolName, len(volNames), volNames, volSizes)
	if err != nil {
		resp <- StorageResult{Error: err}
		return err
	}

	var group = InstanceVolumeGroup{}
	var systemVolume = InstanceVolume{volumes[0], poolName}
	group.System = systemVolume
	group.Snapshots = map[string]ManagedSnapshot{}
	pool.Volumes[systemVolume.Name] = true

	var result = StorageResult{}
	result.Pool = poolName
	result.Volumes = append(result.Volumes, systemVolume.Name)
	if len(volumes) > 1 {
		//data disks
		for _, vol := range volumes[1:] {
			var dataVol = InstanceVolume{vol, poolName}
			group.Data = append(group.Data, dataVol)
			pool.Volumes[vol.Name] = true
			result.Volumes = append(result.Volumes, vol.Name)
		}
	}
	switch bootType {
	case BootTypeCloudInit:
		group.BootImage, err = buildCloudInitImage(manager.initiatorIP, pool.Target, instanceID)
		if err != nil {
			manager.utility.DeleteVolumes(poolName, volNames)
			resp <- StorageResult{Error: err}
			return err
		}
		result.Image = group.BootImage

	default:
		break
	}

	manager.pools[poolName] = pool
	manager.groups[instanceID] = group
	log.Printf("<storage> %d volumes created with group '%s'", len(result.Volumes), instanceID)
	resp <- result
	return manager.saveVolumesMeta(instanceID)
}

func (manager *StorageManager) handleDeleteVolumes(instanceID string, resp chan error) error {
	group, exists := manager.groups[instanceID]
	if !exists {
		err := fmt.Errorf("invalid group '%s'", instanceID)
		resp <- err
		return err
	}
	if group.Locked{
		err := fmt.Errorf("group '%s' is locked for update", instanceID)
		resp <- err
		return err
	}
	//lock for update
	group.Locked = true
	var targets = map[string][]string{group.System.Pool: []string{group.System.Name}}
	for _, vol := range group.Data {
		volumes, exists := targets[vol.Pool]
		if !exists {
			targets[vol.Pool] = []string{vol.Name}
		} else {
			targets[vol.Pool] = append(volumes, vol.Name)
		}
	}
	var poolCount, volCount = 0, 0
	for poolName, vols := range targets {
		if err := manager.utility.DeleteVolumes(poolName, vols); err != nil {
			resp <- err
			return err
		}
		if pool, exists := manager.pools[poolName]; exists {
			for _, volName := range vols {
				delete(pool.Volumes, volName)
			}
		}
		poolCount++
		volCount += len(vols)
	}
	for name, snapshot := range group.Snapshots{
		for volName, volFile := range snapshot.Files{
			if err := os.Remove(volFile); err != nil{
				log.Printf("<storage> warning: remove vol '%s' of snapshot '%s.%s'", volName, instanceID, name)
			}else {
				log.Printf("<storage> vol '%s' of snapshot '%s.%s' removed", volName, instanceID, name)
			}
		}
	}
	if "" != group.BootImage {
		if err := os.Remove(group.BootImage); err != nil {
			log.Printf("<storage> warning:remove boot image '%s' fail: %s", group.BootImage, err.Error())
		} else {
			log.Printf("<storage> boot image '%s' removed", group.BootImage)
		}
	}
	delete(manager.groups, instanceID)
	log.Printf("<storage> %d pools, %d volumes delete with group '%s'", poolCount, volCount, instanceID)
	resp <- nil
	return manager.removeVolumesMeta(instanceID)
}

func (manager *StorageManager) handleReadDiskImage(id framework.SessionID, groupName, targetVol, sourceImage string, targetSize, imageSize uint64,
	mediaHost string, mediaPort uint, startChan chan error, progress chan uint, resultChan chan StorageResult) error {
	group, exists := manager.groups[groupName]
	if !exists {
		err := fmt.Errorf("invalid instance '%s'", groupName)
		startChan <- err
		return err
	}
	if group.Locked{
		err := fmt.Errorf("volume group '%s' locked for update", groupName)
		startChan <- err
		return err
	}
	if group.System.Name != targetVol {
		err := fmt.Errorf("invalid system volume name '%s'", targetVol)
		startChan <- err
		return err
	}
	var path = group.System.Path
	var poolName = group.System.Pool

	if _, exists := manager.tasks[id]; exists {
		err := fmt.Errorf("task %08X not finished", id)
		startChan <- err
		return err
	}
	scheduler, exists := manager.schedulers[poolName]
	if !exists {
		err := fmt.Errorf("no scheduler available for pool '%s'", poolName)
		startChan <- err
		return err
	}
	group.Locked = true
	manager.groups[groupName] = group
	log.Printf("<storage> volume group %s locked for read", groupName)

	scheduler.AddReadTask(id, groupName, targetVol, path, sourceImage, targetSize, imageSize, mediaHost, mediaPort)
	//add task
	manager.tasks[id] = PendingTask{0, progress, resultChan}
	log.Printf("<storage> new read task %08X pending for schedule", id)
	startChan <- nil
	return nil
}

func (manager *StorageManager) handleWriteDiskImage(id framework.SessionID, groupName, targetVol, sourceImage, mediaHost string, mediaPort uint,
	startChan chan error, progress chan uint, resultChan chan StorageResult) error {
	group, exists := manager.groups[groupName]
	if !exists {
		err := fmt.Errorf("invalid instance '%s'", groupName)
		startChan <- err
		return err
	}
	if group.Locked{
		err := fmt.Errorf("volume group '%s' locked for update", groupName)
		startChan <- err
		return err
	}
	if group.System.Name != targetVol {
		err := fmt.Errorf("invalid system volume name '%s'", targetVol)
		startChan <- err
		return err
	}
	var path = group.System.Path
	var poolName = group.System.Pool

	if _, exists := manager.tasks[id]; exists {
		err := fmt.Errorf("task %08X not finished", id)
		startChan <- err
		return err
	}
	scheduler, exists := manager.schedulers[poolName]
	if !exists {
		err := fmt.Errorf("no scheduler available for pool '%s'", poolName)
		startChan <- err
		return err
	}
	group.Locked = true
	manager.groups[groupName] = group
	log.Printf("<storage> volume group %s locked for write", groupName)

	scheduler.AddWriteTask(id, groupName, targetVol, path, sourceImage, mediaHost, mediaPort)
	//add task
	manager.tasks[id] = PendingTask{0, progress, resultChan}
	log.Printf("<storage> new write task %08X pending for schedule", id)
	startChan <- nil
	return nil
}

func (manager *StorageManager) handleResizeVolume(id framework.SessionID, groupName, targetVol string, targetSize uint64, respChan chan StorageResult) error {
	group, exists := manager.groups[groupName]
	var err error
	if !exists {
		err = fmt.Errorf("invalid instance '%s'", groupName)
		respChan <- StorageResult{Error: err}
		return err
	}
	if group.Locked{
		err := fmt.Errorf("volume group '%s' locked for update", groupName)
		respChan <- StorageResult{Error: err}
		return err
	}
	//choose pool&path
	var path, poolName string
	if group.System.Name == targetVol {
		path = group.System.Path
		poolName = group.System.Pool
	} else {
		for _, vol := range group.Data {
			if vol.Name == targetVol {
				path = vol.Path
				poolName = vol.Pool
				break
			}
		}
	}
	if "" == path {
		err = fmt.Errorf("invalid volume '%s'", targetVol)
		respChan <- StorageResult{Error: err}
		return err
	}

	if _, exists := manager.tasks[id]; exists {
		err := fmt.Errorf("previous IO task %08X not finished", id)
		respChan <- StorageResult{Error: err}
		return err
	}
	scheduler, exists := manager.schedulers[poolName]
	if !exists {
		err = fmt.Errorf("no scheduler available for pool '%s'", poolName)
		respChan <- StorageResult{Error: err}
		return err
	}
	group.Locked = true
	manager.groups[groupName] = group
	log.Printf("<storage> volume group %s locked for resize", groupName)
	scheduler.AddResizeTask(id, groupName, targetVol, path, targetSize)
	//add task
	manager.tasks[id] = PendingTask{ResultChan: respChan}
	log.Printf("<storage> new resize task %08X pending for schedule", id)
	return nil
}

func (manager *StorageManager) handleShrinkVolume(id framework.SessionID, groupName, targetVol string, respChan chan StorageResult) error {
	group, exists := manager.groups[groupName]
	var err error
	if !exists {
		err = fmt.Errorf("invalid instance '%s'", groupName)
		respChan <- StorageResult{Error: err}
		return err
	}
	if group.Locked{
		err := fmt.Errorf("volume group '%s' locked for update", groupName)
		respChan <- StorageResult{Error: err}
		return err
	}
	//choose pool&path
	var path, poolName string
	if group.System.Name == targetVol {
		path = group.System.Path
		poolName = group.System.Pool
	} else {
		for _, vol := range group.Data {
			if vol.Name == targetVol {
				path = vol.Path
				poolName = vol.Pool
				break
			}
		}
	}
	if "" == path {
		err = fmt.Errorf("invalid volume '%s'", targetVol)
		respChan <- StorageResult{Error: err}
		return err
	}

	if _, exists := manager.tasks[id]; exists {
		err := fmt.Errorf("previous IO task %08X not finished", id)
		respChan <- StorageResult{Error: err}
		return err
	}
	scheduler, exists := manager.schedulers[poolName]
	if !exists {
		err = fmt.Errorf("no scheduler available for pool '%s'", poolName)
		respChan <- StorageResult{Error: err}
		return err
	}
	group.Locked = true
	manager.groups[groupName] = group
	log.Printf("<storage> volume group %s locked for shrink", groupName)
	scheduler.AddShrinkTask(id, groupName, targetVol, path)
	//add task
	manager.tasks[id] = PendingTask{ResultChan: respChan}
	log.Printf("<storage> new shrink task %08X pending for schedule", id)
	return nil
}

func (manager *StorageManager) handleSchedulerUpdate(update SchedulerUpdate) {
	task, exists := manager.tasks[update.ID]
	if !exists {
		log.Printf("<storage> ignore update for invalid task [%08X]", update.ID)
		return
	}
	task.Progress = update.Progress
	manager.tasks[update.ID] = task
	log.Printf("<storage> update task %08X => %d %%", update.ID, task.Progress)
}

func (manager *StorageManager) handleSchedulerResult(result SchedulerResult) {
	var taskID = result.ID
	task, exists := manager.tasks[taskID]
	if !exists {
		log.Printf("<storage> ignore result for invalid task [%08X]", taskID)
		return
	}
	var err = result.Error
	if err != nil {
		task.ResultChan <- StorageResult{Error: err}
		log.Printf("<storage> schedule task %08X fail: %s", taskID, err.Error())
	} else {
		task.ResultChan <- StorageResult{Size: result.Size}
		log.Printf("<storage> schedule task %08X finished", taskID)
	}
	delete(manager.tasks, taskID)
}

func (manager *StorageManager) handleQuerySnapshot(groupName string, respChan chan StorageResult) (err error){
	var result []SnapshotConfig
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		respChan <- StorageResult{Error:err}
		return err
	}
	if "" == group.BaseSnapshot{
		//no snapshot available
		respChan <- StorageResult{SnapshotList:result}
		return nil
	}
	for _, snapshot := range group.Snapshots{
		result = append(result, snapshot.SnapshotConfig)
	}
	respChan <- StorageResult{SnapshotList:result}
	return nil
}

func (manager *StorageManager) handleGetSnapshot(groupName, snapshotName string, respChan chan StorageResult)  (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		respChan <- StorageResult{Error:err}
		return err
	}
	snapshot, exists := group.Snapshots[snapshotName]
	if !exists{
		err = fmt.Errorf("invalid snapshot '%s'", snapshotName)
		respChan <- StorageResult{Error:err}
		return err
	}
	respChan <- StorageResult{Snapshot:snapshot.SnapshotConfig}
	return nil
}

func (manager *StorageManager) handleCreateSnapshot(groupName, snapshotName, description string, respChan chan error)  (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		respChan <- err
		return err
	}
	if group.Locked{
		err := fmt.Errorf("volume group '%s' locked for update", groupName)
		respChan <- err
		return err
	}
	_, exists = group.Snapshots[snapshotName]
	if exists{
		err = fmt.Errorf("snapshot '%s.%s' exists", groupName, snapshotName)
		respChan <- err
		return err
	}
	var snapshot = ManagedSnapshot{}
	snapshot.Running = false
	snapshot.Name = snapshotName
	snapshot.Description = description
	snapshot.Files = map[string]string{}
	//system volume
	var basePath = filepath.Dir(group.System.Path)

	var targets = map[string]string{}

	var backingSystemPath = filepath.Join(basePath, fmt.Sprintf("%s_%s_sys.%s", groupName, snapshotName, FormatQcow2Suffix))
	snapshot.Files[group.System.Name] = backingSystemPath
	targets[group.System.Path] = backingSystemPath

	for index, volume := range group.Data{
		var backingPath = filepath.Join(basePath, fmt.Sprintf("%s_%s_%d.%s", groupName, snapshotName, index, FormatQcow2Suffix))
		snapshot.Files[volume.Name] = backingPath
		targets[volume.Path] = backingPath
	}

	if nil == group.Snapshots{
		group.Snapshots = map[string]ManagedSnapshot{snapshotName:snapshot}
	}else{
		group.Snapshots[snapshotName] = snapshot
	}
	scheduler, exists := manager.schedulers[group.System.Pool]
	if !exists{
		err = fmt.Errorf("no scheduler for pool '%s'", group.System.Pool)
		respChan <- err
		return err
	}
	group.Locked = true
	manager.groups[groupName] = group
	log.Printf("<storage> volume group '%s' locked for create snapshot", groupName)
	scheduler.AddCreateSnapshotTask(groupName, snapshotName, targets, respChan)
	return nil
}

func (manager *StorageManager) handleDeleteSnapshot(groupName, snapshotName string, respChan chan error)  (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		respChan <- err
		return err
	}
	if group.Locked{
		err := fmt.Errorf("volume group '%s' locked for update", groupName)
		respChan <- err
		return err
	}
	if snapshotName == group.ActiveSnapshot{
		err = errors.New("Not support delete active snapshot")
		respChan <- err
		return err
	}
	if snapshotName == group.BaseSnapshot{
		err = errors.New("Not support delete root snapshot")
		respChan <- err
		return err
	}
	snapshot, exists := group.Snapshots[snapshotName]
	if !exists{
		err = fmt.Errorf("invalid snapshot '%s'", snapshotName)
		respChan <- err
		return err
	}
	for name, s := range group.Snapshots{
		if name == snapshotName{
			continue
		}
		if snapshotName == s.Backing{
			err = fmt.Errorf("snapshot '%s' is depend on '%s', can not delete", name, snapshotName)
			respChan <- err
			return err
		}
	}
	scheduler, exists := manager.schedulers[group.System.Pool]
	if !exists{
		err = fmt.Errorf("no scheduler for pool '%s'", group.System.Pool)
		respChan <- err
		return err
	}
	group.Locked = true
	manager.groups[groupName] = group
	log.Printf("<storage> volume group '%s' locked for delete snapshot", groupName)
	scheduler.AddDeleteSnapshotTask(groupName, snapshotName, snapshot.Files, respChan)
	return nil
}

func (manager *StorageManager) handleRestoreSnapshot(groupName, snapshotName string, respChan chan error)  (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		respChan <- err
		return err
	}
	if group.Locked{
		err := fmt.Errorf("volume group '%s' locked for update", groupName)
		respChan <- err
		return err
	}
	snapshot, exists := group.Snapshots[snapshotName]
	if !exists{
		err = fmt.Errorf("invalid snapshot '%s'", snapshotName)
		respChan <- err
		return err
	}
	//system volume
	var targets = map[string]string{}
	systemBacking, exists := snapshot.Files[group.System.Name]
	if !exists{
		err = fmt.Errorf("no backing file for volume '%s' in snapshot '%s'", group.System.Name, snapshotName)
		respChan <- err
		return err
	}
	targets[group.System.Path] = systemBacking
	//data volume
	for _, volume := range group.Data{
		//todo: new data volume
		backingPath, exists := snapshot.Files[volume.Name]
		if !exists{
			err = fmt.Errorf("no backing file for data volume '%s' in snapshot '%s'", volume.Name, snapshotName)
			respChan <- err
			return err
		}
		targets[volume.Path] = backingPath
	}

	scheduler, exists := manager.schedulers[group.System.Pool]
	if !exists{
		err = fmt.Errorf("no scheduler for pool '%s'", group.System.Pool)
		respChan <- err
		return err
	}
	group.Locked = true
	manager.groups[groupName] = group
	log.Printf("<storage> volume group '%s' locked for restore snapshot", groupName)
	scheduler.AddRestoreSnapshotTask(groupName, snapshotName, targets, respChan)
	return nil
}

func (manager *StorageManager) handleUsingStorage(name, protocol, host, target string, respChan chan StorageResult) (err error){
	switch protocol {
	case StorageProtocolNFS:
		break
	default:
		err = fmt.Errorf("unsupport storage protocol '%s'", protocol)
		respChan <- StorageResult{Error:err}
		return err
	}
	if !manager.nfsEnabled{
		if err = manager.utility.EnableNFSPools(); err != nil{
			log.Printf("<storage> enable nfs pools fail: %s", err.Error())
			respChan <- StorageResult{Error:err}
			return err
		}
		manager.nfsEnabled = true
		log.Println("<storage> nfs pools enabled")
	}
	pool, exists := manager.pools[name]
	if !exists{
		backingPool, err := manager.utility.CreateNFSPool(name, host, target)
		if err != nil{
			log.Printf("<storage> create nfs pool '%s' to %s:%s fail: %s", name, host, target, err.Error())
			respChan <- StorageResult{Error:err}
			return err
		}
		manager.pools[name] = ManagedStoragePool{backingPool, map[string]bool{}, true, ""}
		manager.currentPool = name
		scheduler, err := CreateScheduler(name, manager.progressChan, manager.scheduleChan, manager.eventChan)
		if err != nil {
			log.Printf("<storage> create scheduler for nfs pool '%s' fail: %s", name, err.Error())
			respChan <- StorageResult{Error:err}
			return err
		}
		if err = scheduler.Start();err != nil{
			log.Printf("<storage> start scheduler for nfs pool '%s' fail: %s", name, err.Error())
			respChan <- StorageResult{Error:err}
			return err
		}
		manager.schedulers[name] = scheduler
		respChan <- StorageResult{Path:backingPool.Target}
		log.Printf("<storage> using new nfs pool '%s' to %s:%s", name, host, target)
		return manager.saveConfig()
	}

	//exists
	if name != manager.currentPool{
		//change current pool
		manager.currentPool = name
		respChan <- StorageResult{Path: pool.Target}
		log.Printf("<storage> change current pool to '%s'", name)
		return manager.saveConfig()
	}
	//target changed
	if pool.SourceHost != host || pool.SourceTarget != target{
		if pool.StoragePool, err = manager.utility.ChangeNFSPool(name, host, target); err != nil{
			log.Printf("<storage> change nfs pool '%s' to %s:%s fail: %s", name, host, target, err.Error())
			respChan <- StorageResult{Error:err}
			return err
		}
		manager.pools[name] = pool
		log.Printf("<storage> current nfs pool '%s' changed to %s:%s", name, host, target)
		respChan <- StorageResult{Path: pool.Target}
		return manager.saveConfig()
	}
	mounted, err := manager.utility.IsNFSPoolMounted(name)
	if err != nil{
		log.Printf("<storage> check nfs pool '%s' fail: %s", name, err.Error())
		respChan <- StorageResult{Error:err}
		return err
	}
	if !mounted{
		err = fmt.Errorf("nfs pool '%s' not mounted", name)
		log.Printf("<storage> nfs pool '%s' not ready: %s", name, err.Error())
		respChan <- StorageResult{Error:err}
		return err
	}
	log.Printf("<storage> nfs pool '%s' ready", name)
	respChan <- StorageResult{Path: pool.Target}
	return nil
}

func (manager *StorageManager) handleGetAttachDevices(respChan chan StorageResult) (err error){
	var result []AttachDeviceInfo
	var attachCount, errorCount = 0, 0
	for poolName, pool := range manager.pools{
		if StoragePoolModeLocal == pool.Mode{
			continue
		}
		var device = AttachDeviceInfo{Name:poolName}
		device.Path = pool.Target
		switch pool.Mode {
		case StoragePoolModeNFS:
			device.Protocol = StorageProtocolNFS
			device.Attached = pool.Attached
			if pool.Attached{
				attachCount++
			}else{
				errorCount++
				device.Error = pool.AttachError
			}
			result = append(result, device)
		default:
			log.Printf("<storage> unsupport storage mode %d of pool '%s'", pool.Mode, poolName)
			err = fmt.Errorf("invalid storage mode %d", pool.Mode)
			respChan <- StorageResult{Error:err}
			return err
		}
	}
	if 0 != errorCount{
		log.Printf("<storage> %d device(s) attached, with %d failed device(s)", attachCount, errorCount)
	}else {
		log.Printf("<storage> %d device(s) attached", attachCount)
	}

	respChan <- StorageResult{Devices:result}
	return nil
}

func (manager *StorageManager) handleDetachStorage(respChan chan error)  (err error){
	//detach current nfs pool
	if manager.currentPool == DefaultLocalPoolName{
		log.Printf("<storage> no need to detach local pool '%s'", manager.currentPool)
		respChan <- nil
		return nil
	}
	var poolName = manager.currentPool
	{
		//release scheduler
		scheduler, exists := manager.schedulers[poolName]
		if !exists{
			log.Printf("<storage> warning: no scheduler available for pool '%s'", poolName)
		}else{
			scheduler.Stop()
			delete(manager.schedulers, poolName)
			log.Printf("<storage> scheduler of pool '%s' released", poolName)
		}
	}
	{
		//release pool
		if err = manager.utility.DeleteNFSPool(poolName); err != nil{
			log.Printf("<storage> delete nfs pool '%s' fail: %s", poolName, err.Error())
			respChan <- err
			return nil
		}
		log.Printf("<storage> nfs pool '%s' released", poolName)
		delete(manager.pools, poolName)
		//change to local
		manager.currentPool = DefaultLocalPoolName
		respChan <- nil
	}
	return manager.saveConfig()
}

func (manager *StorageManager) handleAttachVolumeGroup(groups []string, respChan chan error) (err error){
	currentPool, exists := manager.pools[manager.currentPool]
	if !exists {
		err = fmt.Errorf("current pool '%s' not exists", manager.currentPool)
		respChan <- err
		return err
	}
	if currentPool.Mode != StorageModeNFS {
		err = fmt.Errorf("attach not support on non-NFS storage, current pool '%s'", manager.currentPool)
		respChan <- err
		return err
	}
	if err = manager.utility.RefreshPool(manager.currentPool);err != nil{
		respChan <- err
		return err
	}
	for _, groupName := range groups{
		var metaFile = filepath.Join(currentPool.Target, fmt.Sprintf("%s.%s", groupName, VolumeMetaFileSuffix))
		data, err := ioutil.ReadFile(metaFile)
		if err != nil{
			respChan <- err
			return err
		}
		var group InstanceVolumeGroup
		if err = json.Unmarshal(data, &group); err != nil{
			respChan <- err
			return err
		}
		currentPool.Volumes[group.System.Name] = true
		log.Printf("<storage> volume '%s' attached to pool '%s'", group.System.Name, currentPool.Name)
		for _, dataVolume := range group.Data{
			currentPool.Volumes[dataVolume.Name] = true
			log.Printf("<storage> data volume '%s' attached to pool '%s'", dataVolume.Name, currentPool.Name)
		}
		manager.groups[groupName] = group
		log.Printf("<storage> volume group '%s' attached", groupName)
	}
	respChan <- nil
	log.Printf("<storage> %d volume group(s) attached", len(groups))
	return manager.saveConfig()
}

func (manager *StorageManager) handleDetachVolumeGroup(groups []string, respChan chan error) (err error){
	if 0 == len(groups){
		for groupName, _ := range manager.groups{
			groups = append(groups, groupName)
		}
	}
	var targetVolumes = map[string][]string{}
	for _, groupName := range groups{
		group, exists := manager.groups[groupName]
		if !exists{
			err = fmt.Errorf("invalid group '%s'", groupName)
			respChan <- err
			return err
		}
		if group.Locked{
			err = fmt.Errorf("group '%s' locked", groupName)
			respChan <- err
			return err
		}
		pool, exists := manager.pools[group.System.Pool]
		if !exists{
			err = fmt.Errorf("invalid pool '%s' with volume '%s'", group.System.Pool, group.System.Name)
			respChan <- err
			return err
		}
		if pool.Mode != StoragePoolModeNFS{
			err = fmt.Errorf("can not detach from pool '%s' with storage mode %d", pool.Name, pool.Mode)
			respChan <- err
			return err
		}
		volumes, _ := targetVolumes[pool.Name]
		//system volume
		if _, exists = pool.Volumes[group.System.Name]; !exists{
			err = fmt.Errorf("volume '%s' not attached to pool '%s'", group.System.Name, group.System.Pool)
			respChan <- err
			return err
		}
		volumes = append(volumes, group.System.Name)
		//data volumes
		for _, dataVolume := range group.Data{
			if dataVolume.Pool != pool.Name{
				err = fmt.Errorf("volume '%s' not attached in pool '%s'", dataVolume.Name, pool.Name)
				respChan <- err
				return err
			}
			if _, exists = pool.Volumes[dataVolume.Name]; !exists{
				err = fmt.Errorf("volume '%s' not attached to pool '%s'", dataVolume.Name, pool.Name)
				respChan <- err
				return err
			}
			volumes = append(volumes, dataVolume.Name)
		}
		targetVolumes[pool.Name] = volumes
		delete(manager.groups, groupName)
		log.Printf("<storage> volume group '%s' detached", groupName)
	}
	for poolName, volumes := range targetVolumes{
		pool, exists := manager.pools[poolName]
		if !exists{
			err = fmt.Errorf("invalid pool '%s'", poolName)
			respChan <- err
			return err
		}
		for _, volName := range volumes{
			delete(pool.Volumes, volName)
			log.Printf("<storage> volume '%s' detached from pool '%s'", volName, poolName)
		}
		manager.pools[poolName] = pool
	}
	respChan <- nil
	return manager.saveConfig()
}

func (manager *StorageManager) notifyAllTask() {
	for _, task := range manager.tasks {
		if task.ProgressChan != nil {
			task.ProgressChan <- task.Progress
		}
	}
}

func (manager *StorageManager) handleSchedulerEvent(event schedulerEvent) {
	var err error
	switch event.Type {
	case schedulerEventReadDiskCompleted:
		err = manager.handleVolumeTaskCompleted("read", event.Group, event.Volume, event.Error)
	case schedulerEventWriteDiskCompleted:
		err = manager.handleVolumeTaskCompleted("write", event.Group, event.Volume, event.Error)
	case schedulerEventResizeDiskCompleted:
		err = manager.handleVolumeTaskCompleted("resize", event.Group, event.Volume, event.Error)
	case schedulerEventShrinkDiskCompleted:
		err = manager.handleVolumeTaskCompleted("shrink", event.Group, event.Volume, event.Error)
	case schedulerEventCreateSnapshotCompleted:
		err = manager.handleCreateSnapshotCompleted(event.Group, event.Snapshot, event.Error, event.ErrorChan)
	case schedulerEventRestoreSnapshotCompleted:
		err = manager.handleRestoreSnapshotCompleted(event.Group, event.Snapshot, event.Error, event.ErrorChan)
	case schedulerEventDeleteSnapshotCompleted:
		err = manager.handleDeleteSnapshotCompleted(event.Group, event.Snapshot, event.Error, event.ErrorChan)
	default:
		err = fmt.Errorf("unsupported event type %d", event.Type)
	}
	if err != nil{
		log.Printf("<storage> handle scheduler event fail: %s", err.Error())
	}
}

func (manager *StorageManager) handleVolumeTaskCompleted(taskName, groupName, volumeName string, taskError error) (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		return err
	}
	if group.Locked{
		group.Locked = false
		manager.groups[groupName] = group
		log.Printf("<storage> volume group '%s' unlocked for %s complete", groupName, taskName)
	}
	if taskError != nil{
		log.Printf("<storage> warning: %s volume '%s' fail: %s", taskName, volumeName, err.Error())
	//}else{
	//	log.Printf("<storage> debug: %s volume '%s' success", taskName, volumeName)
	}
	return nil
}

func (manager *StorageManager) handleCreateSnapshotCompleted(groupName, snapshotName string, taskError error, errChan chan error) (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		errChan <- err
		return err
	}
	if group.Locked{
		group.Locked = false
		manager.groups[groupName] = group
		log.Printf("<storage> volume group '%s' unlocked for create snapshot complete", groupName)
	}
	snapshot, exists := group.Snapshots[snapshotName]
	if !exists{
		err = fmt.Errorf("invalid snapshot '%s.%s'", groupName, snapshotName)
		errChan <- err
		return err
	}
	if taskError != nil{
		log.Printf("<storage> warning: create snapshot '%s.%s' fail: %s", groupName, snapshotName, err.Error())
		delete(group.Snapshots, snapshotName)
		manager.groups[groupName] = group
		errChan <- taskError
		return nil
	}
	snapshot.IsCurrent = true
	snapshot.CreateTime = time.Now().Format(TimeFormatLayout)
	if "" == group.ActiveSnapshot{
		//no snapshot available, new root
		snapshot.IsRoot = true
		group.ActiveSnapshot = snapshotName
		group.BaseSnapshot = snapshotName
		log.Printf("<storage> new root snapshot '%s.%s' created", groupName, snapshotName)
	}else{
		//backing on current snapshot
		var previousName = group.ActiveSnapshot
		{
			previousSnapshot, exists := group.Snapshots[previousName]
			if !exists{
				log.Printf("<storage> warning: invalid previous snapshot '%s.%s'", groupName, previousName)
			}else{
				previousSnapshot.IsCurrent = false
				group.Snapshots[previousName] = previousSnapshot
			}
		}
		snapshot.Backing = previousName
		group.ActiveSnapshot = snapshotName
		log.Printf("<storage> new snapshot '%s' backing on '%s' created for volume group %s'", snapshotName, snapshot.Backing, groupName)
	}
	group.Snapshots[snapshotName] = snapshot
	manager.groups[groupName] = group
	errChan <- nil
	return manager.saveVolumesMeta(groupName)
}

func (manager *StorageManager) handleRestoreSnapshotCompleted(groupName, snapshotName string, taskError error, errChan chan error) (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		errChan <- err
		return err
	}
	if group.Locked{
		group.Locked = false
		manager.groups[groupName] = group
		log.Printf("<storage> volume group '%s' unlocked for restore snapshot complete", groupName)
	}
	snapshot, exists := group.Snapshots[snapshotName]
	if !exists{
		err = fmt.Errorf("invalid snapshot '%s.%s'", groupName, snapshotName)
		errChan <- err
		return err
	}
	if taskError != nil{
		log.Printf("<storage> warning: restore snapshot '%s.%s' fail: %s", groupName, snapshotName, err.Error())
		errChan <- taskError
		return nil
	}else{
		var previousName = group.ActiveSnapshot
		previous, exists := group.Snapshots[previousName]
		if !exists{
			err = fmt.Errorf("can not find previous snapshot '%s.%s'", groupName, previousName)
			errChan <- nil
			return err
		}
		previous.IsCurrent = false
		snapshot.IsCurrent = true
		group.Snapshots[previousName] = previous
		group.Snapshots[snapshotName] = snapshot
		group.ActiveSnapshot = snapshotName
		manager.groups[groupName] = group
		log.Printf("<stoage> snapshot reverted from '%s.%s' to '%s.%s'", groupName, previousName, groupName, snapshotName)
		errChan <- nil
		return manager.saveVolumesMeta(groupName)
	}
}

func (manager *StorageManager) handleDeleteSnapshotCompleted(groupName, snapshotName string, taskError error, errChan chan error) (err error){
	group, exists := manager.groups[groupName]
	if !exists{
		err = fmt.Errorf("invalid volume group '%s'", groupName)
		errChan <- err
		return err
	}
	if group.Locked{
		group.Locked = false
		manager.groups[groupName] = group
		log.Printf("<storage> volume group '%s' unlocked for delete snapshot complete", groupName)
	}
	if _, exists := group.Snapshots[snapshotName]; !exists{
		err = fmt.Errorf("invalid snapshot '%s.%s'", groupName, snapshotName)
		errChan <- err
		return err
	}
	if taskError != nil{
		log.Printf("<storage> warning: delete snapshot '%s.%s' fail: %s", groupName, snapshotName, err.Error())
	}
	delete(group.Snapshots, snapshotName)
	log.Printf("<storage> snapshot '%s.%s' deleted", groupName, snapshotName)
	errChan <- taskError
	manager.groups[groupName] = group
	return manager.saveVolumesMeta(groupName)
}

func buildCloudInitImage(initiatorIP, poolPath, guestID string) (imagePath string, err error) {
	const (
		NetMode         = "net"
		MetaData        = "meta-data"
		UserData        = "user-data"
		Label           = "cidata"
	)
	var tmpPath = filepath.Join(poolPath, guestID)
	if _, err = os.Stat(tmpPath); os.IsNotExist(err) {
		if err = os.Mkdir(tmpPath, StoragePathPerm); err != nil {
			return
		}
	}
	defer os.RemoveAll(tmpPath)
	var metaFilePath = filepath.Join(tmpPath, MetaData)
	{
		file, err := os.Create(metaFilePath)
		if err != nil {
			return "", err
		}
		//write Meta Data
		fmt.Fprintf(file, "dsmode: %s\n", NetMode)
		var url = fmt.Sprintf("http://%s:%d/latest/%s/\n", initiatorIP, InitiatorMagicPort, guestID)
		fmt.Fprintf(file, "seedfrom: %s", url)
		file.Close()
	}
	var userFilePath = filepath.Join(tmpPath, UserData)
	{
		//empty file
		file, err := os.Create(userFilePath)
		if err != nil {
			return "", err
		}
		file.Close()
	}

	var imageName = fmt.Sprintf("%s_ci.iso", guestID)
	imagePath = filepath.Join(poolPath, imageName)
	var cmd = exec.Command("genisoimage", "-o", imagePath, "-volid", Label, "-joliet", "-rock", metaFilePath, userFilePath)
	if err = cmd.Run(); err != nil {
		return
	}
	log.Printf("<storage> cloud init boot image '%s' created", imagePath)
	return imagePath, nil
}
