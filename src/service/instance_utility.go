package service

import (
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/libvirt/libvirt-go"
	"log"
	"strings"
)

type virDomainOSType struct {
	Name    string `xml:",innerxml"`
	Arch    string `xml:"arch,attr"`
	Machine string `xml:"machine,attr"`
}

type virDomainBootDevice struct {
	Device string `xml:"dev,attr"`
}

type virDomainOSElement struct {
	Type      virDomainOSType       `xml:"type"`
	BootOrder []virDomainBootDevice `xml:"boot"`
	//todo:bootmenu/bootloader/kernal/initrd
}

type virDomainInterfaceTarget struct {
	Device string `xml:"dev,attr,omitempty"`
}

type virDomainInterfaceSource struct {
	Bridge  string `xml:"bridge,attr,omitempty"`
	Network string `xml:"network,attr,omitempty"`
}

type virDomainInterfaceMAC struct {
	Address string `xml:"address,attr"`
}

type virDomainInterfaceModel struct {
	Type string `xml:"type,attr"`
}

type virDomainInterfaceLimit struct {
	Average uint `xml:"average,attr,omitempty"`
	Peak    uint `xml:"peak,attr,omitempty"`
	Burst   uint `xml:"burst,attr,omitempty"`
}

type virDomainInterfaceBandwidth struct {
	Inbound  *virDomainInterfaceLimit `xml:"inbound,omitempty"`
	Outbound *virDomainInterfaceLimit `xml:"outbound,omitempty"`
}

type virDomainInterfaceElement struct {
	XMLName   xml.Name                     `xml:"interface"`
	Type      string                       `xml:"type,attr"`
	Source    virDomainInterfaceSource     `xml:"source,omitempty"`
	MAC       *virDomainInterfaceMAC       `xml:"mac,omitempty"`
	Model     *virDomainInterfaceModel     `xml:"model,omitempty"`
	Target    *virDomainInterfaceTarget    `xml:"target,omitempty"`
	Bandwidth *virDomainInterfaceBandwidth `xml:"bandwidth,omitempty"`
	Filter    *virNwfilterRef
}

type virDomainGraphicsListen struct {
	Type    string `xml:"type,attr,omitempty"`
	Address string `xml:"address,attr,omitempty"`
}

type virDomainGraphicsElement struct {
	XMLName  xml.Name                 `xml:"graphics"`
	Type     string                   `xml:"type,attr"`
	Port     uint                     `xml:"port,attr"`
	Password string                   `xml:"passwd,attr"`
	Listen   *virDomainGraphicsListen `xml:"listen,omitempty"`
}

type virDomainControllerElement struct {
	Type  string `xml:"type,attr"`
	Index string `xml:"index,attr"`
	Model string `xml:"model,attr,omitempty"`
}

type virDomainMemoryStats struct {
	Period uint `xml:"period,attr"`
}

type virDomainMemoryBalloon struct {
	Model string               `xml:"model,attr"`
	Stats virDomainMemoryStats `xml:"stats"`
}

type virDomainChannelTarget struct {
	Type string `xml:"type,attr"`
	Name string `xml:"name,attr"`
}

type virDomainChannel struct {
	Type   string                 `xml:"type,attr"`
	Target virDomainChannelTarget `xml:"target"`
}

type virDomainInput struct {
	Type string `xml:"type,attr"`
	Bus  string `xml:"bus,attr"`
}

type virVideoModel struct {
	Type string `xml:"type,attr"`
}

type virVideoDriver struct {
	Name string `xml:"name,attr"`
}

type virVideoElement struct {
	Model  virVideoModel  `xml:"model"`
	Driver virVideoDriver `xml:"driver"`
}

type virDomainDevicesElement struct {
	Emulator      string                       `xml:"emulator"`
	Disks         []virDomainDiskElement       `xml:"disk,omitempty"`
	Interface     []virDomainInterfaceElement  `xml:"interface,omitempty"`
	Graphics      virDomainGraphicsElement     `xml:"graphics"`
	Controller    []virDomainControllerElement `xml:"controller,omitempty"`
	Input         []virDomainInput             `xml:"input,omitempty"`
	MemoryBalloon virDomainMemoryBalloon       `xml:"memballoon"`
	Channel       virDomainChannel             `xml:"channel"`
	Video         virVideoElement              `xml:"video"`
}

type virDomainCpuElement struct {
	Mode     string                `xml:"mode,attr,omitempty"`
	Match    string                `xml:"match,attr,omitempty"`
	Check    string                `xml:"check,attr,omitempty"`
	Model    []virDomainCpuModel   `xml:"model,omitempty"`
	Features []virDomainCpuFeature `xml:"feature,omitempty"`
	Topology virDomainCpuTopology  `xml:"topology"`
}

type virDomainCpuModel struct {
	Model    string `xml:",innerxml"`
	Fallback string `xml:"fallback,attr,omitempty"`
}

type virDomainCpuFeature struct {
	Policy string `xml:"policy,attr,omitempty"`
	Name   string `xml:"name,attr"`
}

type virDomainCpuTopology struct {
	Sockets uint `xml:"sockets,attr"`
	Cores   uint `xml:"cores,attr"`
	Threads uint `xml:"threads,attr"`
}

type virDomainSuspendToDisk struct {
	Enabled string `xml:"enabled,attr,omitempty"`
}

type virDomainSuspendToMem struct {
	Enabled string `xml:"enabled,attr,omitempty"`
}

type virDomainPowerElement struct {
	Disk virDomainSuspendToDisk `xml:"suspend-to-disk"`
	Mem  virDomainSuspendToMem  `xml:"suspend-to-mem"`
}

type virDomainFeaturePAE struct {
	XMLName xml.Name `xml:"pae"`
}

type virDomainFeatureACPI struct {
	XMLName xml.Name `xml:"acpi"`
}

type virDomainFeatureAPIC struct {
	XMLName xml.Name `xml:"apic"`
}

type virDomainFeatureElement struct {
	PAE  *virDomainFeaturePAE
	ACPI *virDomainFeatureACPI
	APIC *virDomainFeatureAPIC
}

type virDomainClockElement struct {
	Offset string `xml:"offset,attr,omitempty"`
}

type virDomainDiskDriver struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type virDomainDiskSourceHost struct {
	Name string `xml:"name,attr"`
	Port uint   `xml:"port,attr"`
}

type virDomainDiskSource struct {
	File     string                   `xml:"file,attr,omitempty"`
	Protocol string                   `xml:"protocol,attr,omitempty"`
	Name     string                   `xml:"name,attr,omitempty"`
	Pool     string                   `xml:"pool,attr,omitempty"`
	Volume   string                   `xml:"volume,attr,omitempty"`
	Host     *virDomainDiskSourceHost `xml:"host,omitempty"`
}

type virDomainDiskTarget struct {
	Device string `xml:"dev,attr,omitempty"`
	Bus    string `xml:"bus,attr,omitempty"`
}

type virDomainDiskTune struct {
	ReadBytePerSecond  uint `xml:"read_bytes_sec,omitempty"`
	WriteBytePerSecond uint `xml:"write_bytes_sec,omitempty"`
	ReadIOPerSecond    int  `xml:"read_iops_sec,omitempty"`
	WriteIOPerSecond   int  `xml:"write_iops_sec,omitempty"`
}

type virDomainDiskElement struct {
	XMLName  xml.Name             `xml:"disk"`
	Type     string               `xml:"type,attr"`
	Device   string               `xml:"device,attr"`
	Driver   virDomainDiskDriver  `xml:"driver"`
	Target   virDomainDiskTarget  `xml:"target,omitempty"`
	ReadOnly *bool                `xml:"readonly,omitempty"`
	Source   *virDomainDiskSource `xml:"source,omitempty"`
	IoTune   *virDomainDiskTune   `xml:"iotune,omitempty"`
}

type virDomainCPUTuneDefine struct {
	Shares uint `xml:"shares"`
	Period uint `xml:"period"`
	Quota  uint `xml:"quota"`
}

type virDomainDefine struct {
	XMLName     xml.Name                `xml:"domain"`
	Type        string                  `xml:"type,attr"`
	Name        string                  `xml:"name"`
	UUID        string                  `xml:"uuid,omitempty"`
	Memory      uint                    `xml:"memory"` //Default in KiB
	VCpu        uint                    `xml:"vcpu"`
	OS          virDomainOSElement      `xml:"os"`
	CPU         virDomainCpuElement     `xml:"cpu"`
	CPUTune     virDomainCPUTuneDefine  `xml:"cputune,omitempty"`
	Devices     virDomainDevicesElement `xml:"devices,omitempty"`
	OnPowerOff  string                  `xml:"on_poweroff,omitempty"`
	OnReboot    string                  `xml:"on_reboot,omitempty"`
	OnCrash     string                  `xml:"on_crash,omitempty""`
	PowerManage virDomainPowerElement   `xml:"pm,omitempty"`
	Features    virDomainFeatureElement `xml:"features,omitempty"`
	Clock       virDomainClockElement   `xml:"clock"`
}

//nwfilter

type virNwfilerRuleIP struct {
	XMLName         xml.Name `xml:"ip"`
	Protocol        string   `xml:"protocol,attr"`
	SourceAddress   string   `xml:"srcipaddr,attr,omitempty"`
	TargetPortStart uint64   `xml:"dstportstart,attr,omitempty"`
	TargetPortEnd   uint64   `xml:"dstportend,attr,omitempty"`
}

type virNwfilterRule struct {
	XMLName   xml.Name `xml:"rule"`
	Action    string   `xml:"action,attr"`
	Direction string   `xml:"direction,attr"`
	Priority  int      `xml:"priority,attr,omitempty"`
	IPRule    *virNwfilerRuleIP
}

type virNwfilterRef struct {
	XMLName xml.Name `xml:"filterref"`
	Filter  string   `xml:"filter,attr"`
}

type virNwfilterDefine struct {
	XMLName   xml.Name `xml:"filter"`
	Name      string   `xml:"name,attr"`
	UUID      string   `xml:"uuid,omitempty"`
	Reference *virNwfilterRef
	Rules     []virNwfilterRule
}

const (
	IDEOffsetCDROM = iota
	IDEOffsetCIDATA
	IDEOffsetDISK
)

const (
	DiskTypeNetwork        = "network"
	DiskTypeBlock          = "block"
	DiskTypeFile           = "file"
	DiskTypeVolume         = "volume"
	DeviceCDROM            = "cdrom"
	DeviceDisk             = "disk"
	DriverNameQEMU         = "qemu"
	DriverTypeRaw          = "raw"
	DriverTypeQCOW2        = "qcow2"
	StartDeviceCharacter   = 0x61 //'a'
	DevicePrefixIDE        = "hd"
	DevicePrefixSCSI       = "sd"
	DiskBusIDE             = "ide"
	DiskBusSCSI            = "scsi"
	DiskBusSATA            = "sata"
	ProtocolHTTPS          = "https"
	NetworkModelRTL8139    = "rtl8139"
	NetworkModelE1000      = "e1000"
	NetworkModelVIRTIO     = "virtio"
	DisplayDriverVGA       = "vga"
	DisplayDriverCirrus    = "cirrus"
	DisplayDriverQXL       = "qxl"
	DisplayDriverVirtIO    = "virtio"
	DisplayDriverNone      = "none"
	RemoteControlVNC       = "vnc"
	RemoteControlSPICE     = "spice"
	USBModelXHCI           = "nec-xhci"
	USBModelNone           = ""
	TabletBusNone          = ""
	TabletBusVIRTIO        = "virtio"
	TabletBusUSB           = "usb"
	InputTablet            = "tablet"
	PCIController          = "pci"
	DefaultControllerIndex = "0"
	DefaultControllerModel = "pci-root"
	VirtioSCSIController   = "scsi"
	VirtioSCSIModel        = "virtio-scsi"
	USBController          = "usb"
	ListenAllAddress       = "0.0.0.0"
	ListenTypeAddress      = "address"
	NwfilterActionAccept   = "accept"
	NwfilterActionDrop     = "drop"
	NwfilterDirectionIn    = "in"
	NwfilterDirectionOut   = "out"
	NwfilterDirectionInOut = "inout"
	NwfilterPrefix         = "nano-nwfilter-"
	InterfaceTypeBridge    = "bridge"
)

type InstanceUtility struct {
	virConnect *libvirt.Connect
}

func CreateInstanceUtility(connect *libvirt.Connect) (util *InstanceUtility, err error) {
	util = &InstanceUtility{}
	util.virConnect = connect
	return util, nil
}

func (util *InstanceUtility) CreateInstance(config GuestConfig) (guest GuestConfig, err error) {
	var virDomain *libvirt.Domain
	var virNwfilter *libvirt.NWFilter
	defer func() {
		if nil != err {
			if nil != virDomain {
				_ = virDomain.Undefine()
			}
			if nil != virNwfilter {
				_ = virNwfilter.Undefine()
			}
		}
	}()
	var nwfilterDefine = policyToFilter(generateNwfilterName(config.ID), config.ID, config.Security)
	var xmlData []byte
	if xmlData, err = xml.MarshalIndent(nwfilterDefine, "", " "); err != nil {
		err = fmt.Errorf("generate nwfilter define for instance '%s' fail: %s", config.Name, err.Error())
		return
	}
	if virNwfilter, err = util.virConnect.NWFilterDefineXML(string(xmlData)); err != nil {
		err = fmt.Errorf("create nwfilter for instance '%s' fail: %s", config.Name, err.Error())
		return
	}
	var domainDefine virDomainDefine
	if domainDefine, err = util.createDefine(config); err != nil {
		err = fmt.Errorf("create domain define for instance '%s' fail: %s", config.Name, err.Error())
		return
	}

	if xmlData, err = xml.MarshalIndent(domainDefine, "", " "); err != nil {
		err = fmt.Errorf("generete domain define for instance '%s' fail: %s", config.Name, err.Error())
		return
	}
	virDomain, err = util.virConnect.DomainDefineXML(string(xmlData))
	if err != nil {
		log.Printf("debug: failed domain define\n%s\n", string(xmlData))
		err = fmt.Errorf("create domain for instance '%s' fail: %s", config.Name, err.Error())
		return
	}
	if config.AutoStart {
		if err = virDomain.SetAutostart(true); err != nil {
			err = fmt.Errorf("enable auto start for instance '%s' fail: %s", config.Name, err.Error())
			return
		}
	}
	config.Created = true
	//todo: complete return guest
	return config, nil
}

func (util *InstanceUtility) DeleteInstance(id string) (err error) {
	var virDomain *libvirt.Domain
	virDomain, err = util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return
	}
	var running bool
	running, err = virDomain.IsActive()
	if err != nil {
		return
	}
	if running {
		return fmt.Errorf("instance '%s' is running", id)
	}
	var virNwfilter *libvirt.NWFilter
	if virNwfilter, err = util.virConnect.LookupNWFilterByName(generateNwfilterName(id)); err == nil {
		if err = virNwfilter.Undefine(); err != nil {
			return fmt.Errorf("delete nwfilter of instance %s fail: %s", id, err.Error())
		}
	}
	return virDomain.Undefine()
}

func (util *InstanceUtility) Exists(id string) bool {
	_, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return false
	}
	return true
}

func (util *InstanceUtility) GetCPUTimes(id string) (usedNanoseconds uint64, cores uint, err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return
	}
	info, err := virDomain.GetInfo()
	if err != nil {
		return
	}
	return info.CpuTime, info.NrVirtCpu, nil
}

func (util *InstanceUtility) GetIPv4Address(id, mac string) (ip string, err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return
	}
	if !isRunning {
		err = fmt.Errorf("instance '%s' not running", id)
		return
	}
	ifs, err := virDomain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT)
	if err != nil {
		return
	}
	for _, guestInterface := range ifs {
		if guestInterface.Hwaddr == mac {
			if 0 == len(guestInterface.Addrs) {
				err = fmt.Errorf("no address found in interface '%s'", guestInterface.Name)
				return "", err
			}
			for _, addr := range guestInterface.Addrs {
				if libvirt.IP_ADDR_TYPE_IPV4 == libvirt.IPAddrType(addr.Type) {
					ip = addr.Addr
					return ip, nil
				}
			}
		}
	}
	//no hwaddress matched
	return "", nil
}

func (util *InstanceUtility) GetInstanceStatus(id string) (ins InstanceStatus, err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return ins, err
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return ins, err
	}
	ins.Running = isRunning
	if !isRunning {
		return ins, nil
	}
	//running status
	{
		//memory stats
		var statsCount = uint32(libvirt.DOMAIN_MEMORY_STAT_LAST)
		memStats, err := virDomain.MemoryStats(statsCount, 0)
		if err != nil {
			return ins, err
		}
		//size in KB
		var availableValue, rssValue uint64 = 0, 0
		for _, stats := range memStats {
			if stats.Tag == int32(libvirt.DOMAIN_MEMORY_STAT_AVAILABLE) {
				availableValue = stats.Val
				break
			} else if stats.Tag == int32(libvirt.DOMAIN_MEMORY_STAT_RSS) {
				rssValue = stats.Val
			}
		}
		if 0 != availableValue {
			ins.AvailableMemory = availableValue << 10
		} else if 0 != rssValue {
			maxMemory, err := virDomain.GetMaxMemory()
			if err != nil {
				err = fmt.Errorf("get max memory of guest '%s' fail: %s", id, err.Error())
				return ins, err
			}
			if rssValue < maxMemory {
				ins.AvailableMemory = (maxMemory - rssValue) << 10
			} else {
				ins.AvailableMemory = 0
			}
		} else {
			return ins, errors.New("available or rss memory stats not supported")
		}
	}
	desc, err := virDomain.GetXMLDesc(0)
	if err != nil {
		return ins, err
	}
	var define virDomainDefine
	if err = xml.Unmarshal([]byte(desc), &define); err != nil {
		return ins, err
	}

	{
		//disk io
		const (
			DiskTypeVolume = "volume"
		)
		ins.BytesRead = 0
		ins.BytesWritten = 0
		ins.AvailableDisk = 0
		for _, virDisk := range define.Devices.Disks {
			if virDisk.Type != DiskTypeVolume {
				continue
			}
			var devName = virDisk.Target.Device
			//todo: get real available disk space in guest
			//info, err:= virDomain.GetBlockInfo(devName, 0)
			//if err != nil{
			//	return ins, err
			//}
			//ins.AvailableDisk += info.Capacity - info.Allocation
			ins.AvailableDisk = 0
			stats, err := virDomain.BlockStats(devName)
			if err != nil {
				return ins, err
			}
			ins.BytesRead += uint64(stats.RdBytes)
			ins.BytesWritten += uint64(stats.WrBytes)
		}
	}
	{
		//network io
		ins.BytesSent = 0
		ins.BytesReceived = 0
		for index, inf := range define.Devices.Interface {
			if inf.Target == nil {
				return ins, fmt.Errorf("no target available for interface %d", index)
			}

			stats, err := virDomain.InterfaceStats(inf.Target.Device)
			if err != nil {
				return ins, err
			}
			ins.BytesReceived += uint64(stats.RxBytes)
			ins.BytesSent += uint64(stats.TxBytes)
		}
	}
	return ins, nil
}

func (util *InstanceUtility) IsInstanceRunning(id string) (bool, error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return false, err
	}
	return virDomain.IsActive()
}

func (util *InstanceUtility) StartInstance(id string) error {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return err
	}
	if isRunning {
		return fmt.Errorf("instance '%s' already started", id)
	}
	return virDomain.Create()
}

func (util *InstanceUtility) StartInstanceWithMedia(id, host, url string, port uint) error {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return err
	}
	if isRunning {
		return fmt.Errorf("instance '%s' already started", id)
	}

	var ideDevChar = StartDeviceCharacter

	var devName = fmt.Sprintf("%s%c", DevicePrefixIDE, ideDevChar+IDEOffsetCDROM)

	var deviceWithMedia, deviceWithoutMedia string
	var readyOnly = true
	{
		//empty ide cdrom
		var emptyDriver = virDomainDiskElement{Type: DiskTypeBlock, Device: DeviceCDROM, Driver: virDomainDiskDriver{DriverNameQEMU, DriverTypeRaw},
			Target: virDomainDiskTarget{devName, DiskBusIDE}, ReadOnly: &readyOnly}
		if data, err := xml.MarshalIndent(emptyDriver, "", " "); err != nil {
			return err
		} else {
			deviceWithoutMedia = string(data)
		}
	}
	{
		var mediaSource = virDomainDiskSource{Protocol: ProtocolHTTPS, Name: url, Host: &virDomainDiskSourceHost{host, port}}
		var driverWithMedia = virDomainDiskElement{Type: DiskTypeNetwork, Device: DeviceCDROM, Driver: virDomainDiskDriver{DriverNameQEMU, DriverTypeRaw},
			Target: virDomainDiskTarget{devName, DiskBusIDE}, Source: &mediaSource, ReadOnly: &readyOnly}
		if data, err := xml.MarshalIndent(driverWithMedia, "", " "); err != nil {
			return err
		} else {
			deviceWithMedia = string(data)
		}
	}
	//change config before start
	if err = virDomain.UpdateDeviceFlags(deviceWithMedia, libvirt.DOMAIN_DEVICE_MODIFY_CONFIG); err != nil {
		return err
	}
	if err = virDomain.Create(); err != nil {
		return err
	}
	//change live config only
	if err = virDomain.UpdateDeviceFlags(deviceWithoutMedia, libvirt.DOMAIN_DEVICE_MODIFY_CONFIG); err != nil {
		virDomain.Destroy()
		return err
	}
	return nil
}

func (util *InstanceUtility) StopInstance(id string, reboot, force bool) error {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return err
	}
	if !isRunning {
		return fmt.Errorf("instance '%s' already stopped", id)
	}
	//todo: check & unload cdrom before restart
	if reboot {
		if force {
			return virDomain.Reset(0)
		} else {
			return virDomain.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT)
		}
	} else {
		if force {
			return virDomain.Destroy()
		} else {
			return virDomain.Shutdown()
		}
	}
}

func (util *InstanceUtility) InsertMedia(id, host, url string, port uint) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return err
	}
	if !isRunning {
		return fmt.Errorf("instance '%s' stopped", id)
	}
	//assume first ide device as cdrom
	var ideDevChar = StartDeviceCharacter

	var devName = fmt.Sprintf("%s%c", DevicePrefixIDE, ideDevChar+IDEOffsetCDROM)

	var deviceWithMedia string
	{
		var readyOnly = true
		var mediaSource = virDomainDiskSource{Protocol: ProtocolHTTPS, Name: url, Host: &virDomainDiskSourceHost{host, port}}
		var driverWithMedia = virDomainDiskElement{Type: DiskTypeNetwork, Device: DeviceCDROM, Driver: virDomainDiskDriver{DriverNameQEMU, DriverTypeRaw},
			Target: virDomainDiskTarget{devName, DiskBusIDE}, Source: &mediaSource, ReadOnly: &readyOnly}
		if data, err := xml.MarshalIndent(driverWithMedia, "", " "); err != nil {
			return err
		} else {
			deviceWithMedia = string(data)
		}
	}
	//change online device
	return virDomain.UpdateDeviceFlags(deviceWithMedia, libvirt.DOMAIN_DEVICE_MODIFY_LIVE)
}

func (util *InstanceUtility) EjectMedia(id string) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return err
	}
	if !isRunning {
		return fmt.Errorf("instance '%s' stopped", id)
	}
	//assume first ide device as cdrom
	var ideDevChar = StartDeviceCharacter
	var devName = fmt.Sprintf("%s%c", DevicePrefixIDE, ideDevChar+IDEOffsetCDROM)
	var deviceWithoutMedia string
	{
		var readyOnly = true
		//empty ide cdrom
		var emptyDriver = virDomainDiskElement{Type: DiskTypeBlock, Device: DeviceCDROM, Driver: virDomainDiskDriver{DriverNameQEMU, DriverTypeRaw},
			Target: virDomainDiskTarget{devName, DiskBusIDE}, ReadOnly: &readyOnly}
		if data, err := xml.MarshalIndent(emptyDriver, "", " "); err != nil {
			return err
		} else {
			deviceWithoutMedia = string(data)
		}
	}
	//change alive device
	return virDomain.UpdateDeviceFlags(deviceWithoutMedia, libvirt.DOMAIN_DEVICE_MODIFY_LIVE)
}

func (util *InstanceUtility) ModifyCPUTopology(id string, core uint, immediate bool) (err error) {
	const (
		TopologyFormat = "<topology sockets='%d' cores='%d' threads='%d'/>"
	)
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	var currentDomain virDomainDefine
	xmlDesc, err := virDomain.GetXMLDesc(0)
	if err != nil {
		return err
	}
	if err = xml.Unmarshal([]byte(xmlDesc), &currentDomain); err != nil {
		return err
	}
	var previousDefine = fmt.Sprintf(TopologyFormat, currentDomain.CPU.Topology.Sockets,
		currentDomain.CPU.Topology.Cores, currentDomain.CPU.Topology.Threads)
	var newTopology = virDomainCpuTopology{}
	if err = newTopology.SetCpuTopology(core); err != nil {
		return
	}

	var replaceData = fmt.Sprintf(TopologyFormat, newTopology.Sockets,
		newTopology.Cores, newTopology.Threads)

	var modifiedData = strings.Replace(xmlDesc, previousDefine, replaceData, 1)
	if virDomain, err = util.virConnect.DomainDefineXML(modifiedData); err != nil {
		return err
	}
	if err = virDomain.SetVcpusFlags(core, libvirt.DOMAIN_VCPU_CONFIG|libvirt.DOMAIN_VCPU_MAXIMUM); err != nil {
		return err
	}
	return nil
}

func (util *InstanceUtility) ModifyCore(id string, core uint, immediate bool) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	if err = virDomain.SetVcpusFlags(core, libvirt.DOMAIN_VCPU_CONFIG); err != nil {
		return
	}
	return nil
}

func (util *InstanceUtility) ModifyMemory(id string, memory uint, immediate bool) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	var memoryInKiB = uint64(memory >> 10)
	maxMemory, err := virDomain.GetMaxMemory()
	if err != nil {
		return err
	}
	if memoryInKiB > maxMemory {
		if err = virDomain.SetMemoryFlags(memoryInKiB, libvirt.DOMAIN_MEM_CONFIG|libvirt.DOMAIN_MEM_MAXIMUM); err != nil {
			return
		}
	}
	if err = virDomain.SetMemoryFlags(memoryInKiB, libvirt.DOMAIN_MEM_CONFIG); err != nil {
		return
	}
	return nil
}

func (util *InstanceUtility) ModifyPassword(id, user, password string) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(id)
	if err != nil {
		return err
	}
	err = virDomain.SetUserPassword(user, password, 0)
	return err
}

func (util *InstanceUtility) ModifyAutoStart(guestID string, enable bool) (err error) {
	var virDomain *libvirt.Domain
	if virDomain, err = util.virConnect.LookupDomainByUUIDString(guestID); err != nil {
		err = fmt.Errorf("get guest fail: %s", err.Error())
		return
	}
	var current bool
	if current, err = virDomain.GetAutostart(); err != nil {
		err = fmt.Errorf("check auto start status fail: %s", err.Error())
		return
	}
	if current == enable {
		if current {
			err = fmt.Errorf("auto start of guest '%s' already enabled", guestID)
		} else {
			err = fmt.Errorf("auto start of guest '%s' already disabled", guestID)
		}
		return
	}
	if err = virDomain.SetAutostart(enable); err != nil {
		err = fmt.Errorf("set auto start fail: %s", err.Error())
		return
	}
	return
}

func (util *InstanceUtility) SetCPUThreshold(guestID string, priority PriorityEnum) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(guestID)
	if err != nil {
		return err
	}
	var xmlContent string
	xmlContent, err = virDomain.GetXMLDesc(0)
	if err != nil {
		return
	}
	var newDefine virDomainDefine
	if err = xml.Unmarshal([]byte(xmlContent), &newDefine); err != nil {
		return
	}
	if err = setCPUPriority(&newDefine, priority); err != nil {
		return
	}
	data, err := xml.MarshalIndent(newDefine, "", " ")
	if err != nil {
		return err
	}
	_, err = util.virConnect.DomainDefineXML(string(data))
	if err != nil {
		err = fmt.Errorf("define fail: %s, content: %s", err.Error(), string(data))
	}
	return
}

func setCPUPriority(domain *virDomainDefine, priority PriorityEnum) (err error) {
	const (
		periodPerSecond = 1000000
		quotaPerSecond  = 1000000
		highShares      = 2000
		mediumShares    = 1000
		lowShares       = 500
	)
	domain.CPUTune.Period = periodPerSecond
	switch priority {
	case PriorityHigh:
		domain.CPUTune.Shares = highShares
		domain.CPUTune.Quota = quotaPerSecond
		break
	case PriorityMedium:
		domain.CPUTune.Shares = mediumShares
		domain.CPUTune.Quota = quotaPerSecond / 2
		break
	case PriorityLow:
		domain.CPUTune.Shares = lowShares
		domain.CPUTune.Quota = quotaPerSecond / 4
		break
	default:
		return fmt.Errorf("invalid CPU priority %d", priority)
	}
	return nil
}

func (util *InstanceUtility) SetDiskThreshold(guestID string, writeSpeed, writeIOPS, readSpeed, readIOPS uint64) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(guestID)
	if err != nil {
		return err
	}
	var currentDomain virDomainDefine
	xmlDesc, err := virDomain.GetXMLDesc(0)
	if err != nil {
		return err
	}
	if err = xml.Unmarshal([]byte(xmlDesc), &currentDomain); err != nil {
		return err
	}
	var activated bool
	if activated, err = virDomain.IsActive(); err != nil {
		return err
	}
	if activated {
		err = fmt.Errorf("instance %s ('%s') is still running", currentDomain.Name, guestID)
		return
	}
	//var affectFlag = libvirt.DOMAIN_AFFECT_CONFIG | libvirt.DOMAIN_AFFECT_LIVE
	//deprecated: unsupported configuration: the block I/O throttling group parameter is not supported with this QEMU binary
	//var affectFlag = libvirt.DOMAIN_AFFECT_CONFIG
	//var parameters libvirt.DomainBlockIoTuneParameters
	//parameters.WriteIopsSec = writeIOPS
	//parameters.WriteIopsSecSet = true
	//parameters.ReadIopsSec = readIOPS
	//parameters.ReadIopsSecSet = true
	//parameters.GroupNameSet = true
	//parameters.GroupName = "default"
	//
	//for _, dev := range currentDomain.Devices.Disks {
	//	if nil != dev.ReadOnly {
	//		//ignore readonly device
	//		continue
	//	}
	//	if err = virDomain.SetBlockIoTune(dev.Target.Device, &parameters, affectFlag); err != nil {
	//		err = fmt.Errorf("set block io tune fail for dev '%s' : %s", dev.Target.Device, err.Error())
	//		return
	//	}
	//}
	//return nil

	var impactFlag = libvirt.DOMAIN_DEVICE_MODIFY_CONFIG

	for _, dev := range currentDomain.Devices.Disks {
		if nil != dev.ReadOnly {
			//ignore readonly device
			continue
		}
		//static configure
		var tune = dev.IoTune
		if tune == nil {
			tune = &virDomainDiskTune{}
		}
		tune.WriteIOPerSecond = int(writeIOPS)
		tune.ReadIOPerSecond = int(readIOPS)
		tune.WriteBytePerSecond = uint(writeSpeed)
		tune.ReadBytePerSecond = uint(readSpeed)

		dev.IoTune = tune
		data, err := xml.MarshalIndent(dev, "", " ")
		if err != nil {
			return err
		}
		if err = virDomain.UpdateDeviceFlags(string(data), impactFlag); err != nil {
			err = fmt.Errorf("update device fail: %s, content: %s", err.Error(), string(data))
			return err
		}
	}
	return nil
}

func (util *InstanceUtility) SetNetworkThreshold(guestID string, receiveSpeed, sendSpeed uint64) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(guestID)
	if err != nil {
		return err
	}
	var currentDomain virDomainDefine
	xmlDesc, err := virDomain.GetXMLDesc(0)
	if err != nil {
		return err
	}
	if err = xml.Unmarshal([]byte(xmlDesc), &currentDomain); err != nil {
		return err
	}
	var activated bool
	if activated, err = virDomain.IsActive(); err != nil {
		return err
	}
	//var affectFlag = libvirt.DOMAIN_AFFECT_CONFIG | libvirt.DOMAIN_AFFECT_LIVE
	var affectFlag = libvirt.DOMAIN_AFFECT_LIVE
	//API params
	var parameters libvirt.DomainInterfaceParameters
	if activated {
		if 0 != receiveSpeed {
			parameters.BandwidthInAverageSet = true
			parameters.BandwidthInAverage = uint(receiveSpeed >> 10) //kbytes
			parameters.BandwidthInPeakSet = true
			parameters.BandwidthInPeak = uint(receiveSpeed >> 9) // average * 2
			parameters.BandwidthInBurstSet = true
			parameters.BandwidthInBurst = uint(receiveSpeed >> 10)
		}
		if 0 != sendSpeed {
			parameters.BandwidthOutAverageSet = true
			parameters.BandwidthOutAverage = uint(sendSpeed >> 10) //kbytes
			parameters.BandwidthOutPeakSet = true
			parameters.BandwidthOutPeak = uint(sendSpeed >> 9) // average * 2
			parameters.BandwidthOutBurstSet = true
			parameters.BandwidthOutBurst = uint(sendSpeed >> 10)
		}
	}
	//configure params
	var impactFlag = libvirt.DOMAIN_DEVICE_MODIFY_CONFIG
	var bandWidth = virDomainInterfaceBandwidth{}
	if 0 != receiveSpeed {
		var inbound = virDomainInterfaceLimit{uint(receiveSpeed >> 10), uint(receiveSpeed >> 9), uint(receiveSpeed >> 10)}
		bandWidth.Inbound = &inbound
	}
	if 0 != sendSpeed {
		var outbound = virDomainInterfaceLimit{uint(sendSpeed >> 10), uint(sendSpeed >> 9), uint(sendSpeed >> 10)}
		bandWidth.Outbound = &outbound
	}

	for _, netInf := range currentDomain.Devices.Interface {
		if activated {
			if err = virDomain.SetInterfaceParameters(netInf.Target.Device, &parameters, affectFlag); err != nil {
				return
			}
		}
		netInf.Bandwidth = &bandWidth
		data, err := xml.MarshalIndent(netInf, "", " ")
		if err != nil {
			return err
		}
		if err = virDomain.UpdateDeviceFlags(string(data), impactFlag); err != nil {
			err = fmt.Errorf("update device fail: %s, content: %s", err.Error(), string(data))
			return err
		}
	}
	return nil
}

func (util *InstanceUtility) Rename(uuid, newName string) (err error) {
	virDomain, err := util.virConnect.LookupDomainByUUIDString(uuid)
	if err != nil {
		return err
	}
	isRunning, err := virDomain.IsActive()
	if err != nil {
		return err
	}
	if isRunning {
		return fmt.Errorf("instance '%s' is still running", uuid)
	}
	var currentName string
	currentName, err = virDomain.GetName()
	if err != nil {
		return
	}
	if currentName == newName {
		return errors.New("no need to change")
	}
	return virDomain.Rename(newName, 0)
}

func (util *InstanceUtility) ResetMonitorSecret(uuid string, display string, port uint, secret string) (err error) {
	var virDomain *libvirt.Domain
	if virDomain, err = util.virConnect.LookupDomainByUUIDString(uuid); err != nil {
		err = fmt.Errorf("invalid guest '%s'", uuid)
		return err
	}
	var isRunning bool
	if isRunning, err = virDomain.IsActive(); err != nil {
		err = fmt.Errorf("check running status fail: %s", err.Error())
		return err
	}
	var updateFlag = libvirt.DOMAIN_DEVICE_MODIFY_CONFIG
	if isRunning {
		updateFlag |= libvirt.DOMAIN_DEVICE_MODIFY_LIVE
	}
	var graphics = virDomainGraphicsElement{
		Type:     display,
		Port:     port,
		Password: secret,
		Listen:   &virDomainGraphicsListen{ListenTypeAddress, ListenAllAddress},
	}
	var payload []byte
	if payload, err = xml.Marshal(graphics); err != nil {
		err = fmt.Errorf("generate graphic element fail: %s", err.Error())
		return
	}
	if err = virDomain.UpdateDeviceFlags(string(payload), updateFlag); err != nil {
		err = fmt.Errorf("update graphic element fail: %s", err.Error())
		return
	}
	return nil
}

func (util *InstanceUtility) InitialDomainNwfilter(instanceID string, policy SecurityPolicy) (err error) {
	var filterName = generateNwfilterName(instanceID)
	var xmlData []byte
	var xmlString string
	var filterDefine = policyToFilter(filterName, instanceID, &policy)
	if xmlData, err = xml.MarshalIndent(filterDefine, "", " "); err != nil {
		err = fmt.Errorf("generate nwfilter xml for instance '%s' fail: %s", instanceID, err.Error())
		return
	}
	if _, err = util.virConnect.NWFilterDefineXML(string(xmlData)); err != nil {
		err = fmt.Errorf("create nwfilter for guest '%s' fail: %s", instanceID, err.Error())
		return
	}
	{
		var virDomain *libvirt.Domain
		if virDomain, err = util.virConnect.LookupDomainByUUIDString(instanceID); err != nil {
			err = fmt.Errorf("invalid guest id '%s'", instanceID)
			return
		}

		if xmlString, err = virDomain.GetXMLDesc(0); err != nil {
			err = fmt.Errorf("get xml of guest '%s' fail: %s", instanceID, err.Error())
			return
		}
		var domainDefine virDomainDefine
		if err = xml.Unmarshal([]byte(xmlString), &domainDefine); err != nil {
			err = fmt.Errorf("unmarshal xml of guest '%s' fail: %s", instanceID, err.Error())
			return
		}
		var isActive bool
		if isActive, err = virDomain.IsActive(); err != nil {
			err = fmt.Errorf("check running status of guest '%s' fail: %s", instanceID, err.Error())
			return
		}
		var updateFlag = libvirt.DOMAIN_DEVICE_MODIFY_CONFIG
		if isActive {
			updateFlag |= libvirt.DOMAIN_DEVICE_MODIFY_LIVE
		}
		for _, interfaceDefine := range domainDefine.Devices.Interface {
			if InterfaceTypeBridge == interfaceDefine.Type {
				if nil == interfaceDefine.Filter {
					var deviceWithFilter = interfaceDefine
					deviceWithFilter.Filter = &virNwfilterRef{
						Filter: filterName,
					}
					if xmlData, err = xml.MarshalIndent(deviceWithFilter, "", " "); err != nil {
						err = fmt.Errorf("marshal interface device of guest '%s' fail: %s", instanceID, err.Error())
						return
					}
					xmlString = string(xmlData)
					if err = virDomain.UpdateDeviceFlags(xmlString, updateFlag); err != nil {
						err = fmt.Errorf("update nwfilter to interface of guest '%s' fail: %s", instanceID, err.Error())
					}
				}
			}
		}
	}

	return
}

func (util *InstanceUtility) SyncDomainNwfilter(id string, policy *SecurityPolicy) (err error) {
	var filterName = generateNwfilterName(id)
	var filterDefine = policyToFilter(filterName, id, policy)
	var xmlData []byte
	if xmlData, err = xml.MarshalIndent(filterDefine, "", " "); err != nil {
		err = fmt.Errorf("generate nwfilter xml for instance '%s' fail: %s", id, err.Error())
		return
	}
	_, err = util.virConnect.NWFilterDefineXML(string(xmlData))
	return
}

func (util *InstanceUtility) createDefine(config GuestConfig) (define virDomainDefine, err error) {
	const (
		BootDeviceCDROM    = "cdrom"
		BootDeviceHardDisk = "hd"
		cpuModeCustom      = "custom"
		cpuMatchExact      = "exact"
		cpuMatchMinimum    = "minimum"
		cpuCheckPartial    = "partial"
		cpuCheckFull       = "full"
		modelFallbackAllow = "allow"
		modelCore2Duo      = "core2duo"
		cpuFeatureMonitor  = "monitor"
		// policy: disable, require, optional, force, forbid
		cpuFeaturePolicyDisable  = "disable"
		cpuFeaturePolicyRequire  = "require"
		cpuFeaturePolicyOptional = "optional"
		cpuFeaturePolicyForce    = "force"
		cpuFeaturePolicyForbid   = "forbid"
	)
	const (
		modelIntelHaswell = "Haswell"
		modelAMDOpteronG5 = "Opteron_G5"
		modelDefault      = modelIntelHaswell
	)

	define.Initial()
	define.Name = config.Name
	define.UUID = config.ID
	define.Memory = config.Memory >> 10
	define.VCpu = config.Cores

	//for windows 2016 compatibility
	//<cpu mode='custom' match='exact' check='partial'>
	//<model fallback='allow'>core2duo</model>
	//<feature name='monitor' policy='disable'/>
	//<topology sockets='1' cores='1' threads='1'/>
	//</cpu>
	define.CPU.Mode = cpuModeCustom
	define.CPU.Match = cpuMatchMinimum
	define.CPU.Check = cpuCheckPartial
	define.CPU.Model = []virDomainCpuModel{
		{
			Model:    modelDefault,
			Fallback: modelFallbackAllow,
		},
	}
	{
		//CPU features
		var forced = []string{
			"acpi",
			"apic",
			"pae",
		}
		var disabled = []string{
			cpuFeatureMonitor,
			"hle",
			"rtm",
		}
		var preferedIntel = []string{
			"pcid",
			"spec-ctrl",
			"stibp",
			"ssbd",
			"pdpe1gb",
			"md-clear",
		}
		var preferedAMD = []string{
			"ibpb",
			"stibp",
			"virt-ssbd",
			"pdpe1gb",
		}
		//merge optional features
		// feature => true
		var prefered = map[string]bool{}
		for _, feature := range preferedIntel {
			prefered[feature] = true
		}
		for _, feature := range preferedAMD {
			prefered[feature] = true
		}
		var features = make([]virDomainCpuFeature, 0)
		for _, feature := range forced {
			features = append(features, virDomainCpuFeature{
				Name:   feature,
				Policy: cpuFeaturePolicyForce,
			})
		}
		for _, feature := range disabled {
			features = append(features, virDomainCpuFeature{
				Name:   feature,
				Policy: cpuFeaturePolicyDisable,
			})
		}
		for feature := range prefered {
			features = append(features, virDomainCpuFeature{
				Name:   feature,
				Policy: cpuFeaturePolicyOptional,
			})
		}
		define.CPU.Features = features
	}

	//cpu
	if err = setCPUPriority(&define, config.CPUPriority); err != nil {
		err = fmt.Errorf("set CPU prioirity fail: %s", err.Error())
		return
	}
	if err = define.CPU.Topology.SetCpuTopology(config.Cores); err != nil {
		err = fmt.Errorf("set CPU topology fail: %s", err.Error())
		return
	}
	define.OS.BootOrder = []virDomainBootDevice{
		{Device: BootDeviceCDROM},
		{Device: BootDeviceHardDisk},
	}
	define.SetVideoDriver(config.Template.Display)

	switch config.StorageMode {
	case StorageModeLocal:
		if err = define.SetLocalVolumes(config.Template.Disk, config.StoragePool, config.StorageVolumes, config.BootImage,
			config.ReadSpeed, config.ReadIOPS, config.WriteSpeed, config.WriteIOPS); err != nil {
			err = fmt.Errorf("set local volumes fail: %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("unsupported storage mode :%d", config.StorageMode)
		return
	}
	var nwfilterName = generateNwfilterName(config.ID)
	switch config.NetworkMode {
	case NetworkModePlain:
		if err = define.SetPlainNetwork(config.Template.Network, config.NetworkSource, config.HardwareAddress, nwfilterName,
			config.ReceiveSpeed, config.SendSpeed); err != nil {
			err = fmt.Errorf("set plain network fail: %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("unsupported network mode :%d", config.NetworkMode)
		return
	}
	if err = define.SetRemoteControl(config.Template.Control, config.MonitorPort, config.MonitorSecret); err != nil {
		err = fmt.Errorf("set remote control fail: %s", err.Error())
		return
	}
	//tablet
	if config.Template.Tablet != TabletBusNone {
		define.Devices.Input = append(define.Devices.Input, virDomainInput{InputTablet, config.Template.Tablet})
	}
	if config.Template.USB != USBModelNone {
		define.Devices.Controller = append(define.Devices.Controller, virDomainControllerElement{USBController, DefaultControllerIndex, config.Template.USB})
	}
	return
}

func (topology *virDomainCpuTopology) SetCpuTopology(totalThreads uint) error {
	const (
		//SplitThreshold = 4
		ThreadPerCore = 2
		MaxCores      = 1 << 5
		MaxSockets    = 1 << 3
	)
	if (totalThreads > 1) && (0 != (totalThreads % 2)) {
		return fmt.Errorf("even core number ( %d ) is not allowed", totalThreads)
	}
	var threads, cores, sockets uint
	if totalThreads < ThreadPerCore {
		threads = 1
		sockets = 1
		cores = totalThreads
	} else {
		threads = ThreadPerCore
		cores = totalThreads / threads
		sockets = 1
		if cores > MaxCores {
			for sockets = 2; sockets < MaxSockets+1; sockets = sockets << 1 {
				cores = (totalThreads / threads) / sockets
				if cores <= MaxCores {
					break
				}
			}
			if cores > MaxCores {
				return fmt.Errorf("no proper cpu topology fit total threads %d", totalThreads)
			}
		}
	}
	topology.Threads = threads
	topology.Cores = cores
	topology.Sockets = sockets
	return nil
}

func (define *virDomainDefine) SetVideoDriver(model string) {
	define.Devices.Video.Model.Type = model
}

func (define *virDomainDefine) SetLocalVolumes(diskBus, pool string, volumes []string, bootImage string,
	readSpeed, readIOPS, writeSpeed, writeIOPS uint64) error {
	var readyOnly = true
	{
		//empty ide cdrom
		var devName = fmt.Sprintf("%s%c", DevicePrefixIDE, StartDeviceCharacter+IDEOffsetCDROM)
		var cdromElement = virDomainDiskElement{Type: DiskTypeBlock, Device: DeviceCDROM, Driver: virDomainDiskDriver{DriverNameQEMU, DriverTypeRaw},
			Target: virDomainDiskTarget{devName, DiskBusIDE}, ReadOnly: &readyOnly}
		define.Devices.Disks = append(define.Devices.Disks, cdromElement)
	}
	if "" != bootImage {
		//cdrom for ci data
		var ciDevice = fmt.Sprintf("%s%c", DevicePrefixIDE, StartDeviceCharacter+IDEOffsetCIDATA)
		var isoSource = virDomainDiskSource{File: bootImage}
		var ciElement = virDomainDiskElement{Type: DiskTypeFile, Device: DeviceCDROM, Driver: virDomainDiskDriver{DriverNameQEMU, DriverTypeRaw},
			Target: virDomainDiskTarget{ciDevice, DiskBusIDE}, Source: &isoSource, ReadOnly: &readyOnly}
		define.Devices.Disks = append(define.Devices.Disks, ciElement)
	}
	var devicePrefix string
	var devChar int
	if DiskBusIDE == diskBus {
		//ide device
		devChar = StartDeviceCharacter + IDEOffsetDISK
		devicePrefix = DevicePrefixIDE
	} else {
		//sata/scsi
		devChar = StartDeviceCharacter
		devicePrefix = DevicePrefixSCSI
	}

	var ioTune *virDomainDiskTune
	if 0 != writeSpeed || 0 != writeIOPS || 0 != readSpeed || 0 != readIOPS {
		var limit = virDomainDiskTune{}
		limit.ReadBytePerSecond = uint(readSpeed)
		limit.ReadIOPerSecond = int(readIOPS)
		limit.WriteBytePerSecond = uint(writeSpeed)
		limit.WriteIOPerSecond = int(writeIOPS)
		ioTune = &limit
	}

	for _, volumeName := range volumes {
		var devName = fmt.Sprintf("%s%c", devicePrefix, devChar)
		var source = virDomainDiskSource{Pool: pool, Volume: volumeName}
		var diskElement = virDomainDiskElement{Type: DiskTypeVolume, Device: DeviceDisk, Driver: virDomainDiskDriver{DriverNameQEMU, DriverTypeQCOW2},
			Target: virDomainDiskTarget{devName, diskBus}, Source: &source}
		if ioTune != nil {
			diskElement.IoTune = ioTune
		}
		define.Devices.Disks = append(define.Devices.Disks, diskElement)
		devChar++
	}
	return nil
}

func (define *virDomainDefine) SetPlainNetwork(netBus, bridge, mac, filterName string, receiveSpeed, sendSpeed uint64) (err error) {
	if mac == "" {
		mac, err = define.generateMacAddress()
		if err != nil {
			return err
		}
	}
	var i = virDomainInterfaceElement{}
	i.Type = InterfaceTypeBridge
	i.MAC = &virDomainInterfaceMAC{mac}
	i.Source = virDomainInterfaceSource{Bridge: bridge}
	i.Model = &virDomainInterfaceModel{netBus}
	i.Filter = &virNwfilterRef{Filter: filterName}
	if 0 != receiveSpeed || 0 != sendSpeed {
		var bandWidth = virDomainInterfaceBandwidth{}
		if 0 != receiveSpeed {
			var limit = virDomainInterfaceLimit{}
			limit.Average = uint(receiveSpeed >> 10)
			limit.Peak = uint(receiveSpeed >> 9)
			limit.Burst = limit.Average
			bandWidth.Inbound = &limit
		}
		if 0 != sendSpeed {
			var limit = virDomainInterfaceLimit{}
			limit.Average = uint(sendSpeed >> 10)
			limit.Peak = uint(sendSpeed >> 9)
			limit.Burst = limit.Average
			bandWidth.Outbound = &limit
		}
		i.Bandwidth = &bandWidth
	}
	define.Devices.Interface = []virDomainInterfaceElement{i}
	return nil
}

func (define *virDomainDefine) generateMacAddress() (string, error) {
	const (
		BufferSize = 3
		MacPrefix  = "00:16:3e"
	)
	buf := make([]byte, BufferSize)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%02x:%02x:%02x", MacPrefix, buf[0], buf[1], buf[2]), nil
}

func (define *virDomainDefine) SetRemoteControl(display string, port uint, secret string) error {
	define.Devices.Graphics.Port = port
	define.Devices.Graphics.Type = display
	define.Devices.Graphics.Password = secret
	define.Devices.Graphics.Listen = &virDomainGraphicsListen{ListenTypeAddress, ListenAllAddress}
	return nil
}

func (define *virDomainDefine) Initial() {
	const (
		KVMInstanceType     = "kvm"
		DefaultOSName       = "hvm"
		DefaultOSArch       = "x86_64"
		DefaultOSMachine    = "pc"
		DefaultOSMachineQ35 = "q35"
		DefaultQEMUEmulator = "/usr/bin/qemu-system-x86_64"
		DestroyInstance     = "destroy"
		RestartInstance     = "restart"
		DefaultPowerEnabled = "no"
		DefaultClockOffset  = "utc"

		MemoryBalloonModel  = "virtio"
		MemoryBalloonPeriod = 2

		ChannelType       = "unix"
		ChannelTargetType = "virtio"
		ChannelTargetName = "org.qemu.guest_agent.0"
	)
	define.Type = KVMInstanceType
	define.OS.Type.Name = DefaultOSName
	define.OS.Type.Arch = DefaultOSArch
	define.OS.Type.Machine = DefaultOSMachine
	//define.OS.Type.Machine = DefaultOSMachineQ35
	define.Devices.Emulator = DefaultQEMUEmulator

	define.OnPowerOff = DestroyInstance
	define.OnReboot = RestartInstance
	define.OnCrash = RestartInstance

	define.PowerManage.Disk.Enabled = DefaultPowerEnabled
	define.PowerManage.Mem.Enabled = DefaultPowerEnabled

	define.Features.PAE = &virDomainFeaturePAE{}
	define.Features.ACPI = &virDomainFeatureACPI{}
	define.Features.APIC = &virDomainFeatureAPIC{}

	define.Clock.Offset = DefaultClockOffset

	define.Devices.MemoryBalloon.Model = MemoryBalloonModel
	define.Devices.MemoryBalloon.Stats.Period = MemoryBalloonPeriod

	define.Devices.Channel.Type = ChannelType
	define.Devices.Channel.Target = virDomainChannelTarget{ChannelTargetType, ChannelTargetName}

	define.Devices.Controller = []virDomainControllerElement{
		virDomainControllerElement{Type: PCIController, Index: DefaultControllerIndex, Model: DefaultControllerModel},
		virDomainControllerElement{Type: VirtioSCSIController, Index: DefaultControllerIndex, Model: VirtioSCSIModel}}

}

func policyToFilter(name, uuid string, policy *SecurityPolicy) (nwfilter virNwfilterDefine) {
	const LowestPriority = 1000
	if nil == policy {
		//accept by default
		policy = &SecurityPolicy{Accept: true}
	}
	nwfilter.Name = name
	nwfilter.UUID = uuid
	var priority = LowestPriority - len(policy.Rules) - 1
	var outRule = virNwfilterRule{
		Direction: NwfilterDirectionOut,
		Priority:  priority,
		Action:    NwfilterActionAccept,
	}
	nwfilter.Rules = append(nwfilter.Rules, outRule)
	priority++
	for _, rule := range policy.Rules {
		var virRule = virNwfilterRule{
			Direction: NwfilterDirectionIn,
			Priority:  priority,
		}
		if rule.Accept {
			virRule.Action = NwfilterActionAccept
		} else {
			virRule.Action = NwfilterActionDrop
		}
		virRule.IPRule = &virNwfilerRuleIP{
			Protocol:        string(rule.Protocol),
			TargetPortStart: uint64(rule.TargetPort),
			SourceAddress:   rule.SourceAddress,
		}
		nwfilter.Rules = append(nwfilter.Rules, virRule)
		priority++
	}
	var defaultRule = virNwfilterRule{
		Direction: NwfilterDirectionIn,
		Priority:  priority,
	}
	if policy.Accept {
		defaultRule.Action = NwfilterActionAccept
	} else {
		defaultRule.Action = NwfilterActionDrop
	}
	nwfilter.Rules = append(nwfilter.Rules, defaultRule)
	return
}

func generateNwfilterName(id string) string {
	return NwfilterPrefix + id
}
