package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"log"
)

type HandleComputePoolReadyExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
	StorageModule  service.StorageModule
	NetworkModule  service.NetworkModule
}

func (executor *HandleComputePoolReadyExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	storageName, err := request.GetString(framework.ParamKeyStorage)
	if err != nil{
		return
	}
	networkName, err := request.GetString(framework.ParamKeyNetwork)
	if err != nil{
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.ComputeCellReadyEvent)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	if "" == storageName{
		log.Printf("[%08X] recv compute pool '%s' ready from %s", id, poolName, request.GetSender())
		//try detach
		var respChan = make(chan error, 1)
		executor.StorageModule.DetachStorage(respChan)
		err = <- respChan
		if err != nil{
			resp.SetError(err.Error())
			log.Printf("[%08X] detach storage fail: %s", id, err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}else{
		var protocol, host, target string
		if protocol, err = request.GetString(framework.ParamKeyType); err != nil{
			return
		}
		if host, err = request.GetString(framework.ParamKeyHost); err != nil{
			return
		}
		if target, err = request.GetString(framework.ParamKeyTarget); err != nil{
			return
		}
		log.Printf("[%08X] recv compute pool '%s' ready using storage '%s' from %s", id, poolName, storageName, request.GetSender())
		var storageURL string
		{
			var respChan = make(chan service.StorageResult, 1)
			executor.StorageModule.UsingStorage(storageName, protocol, host, target, respChan)
			var result = <- respChan
			if result.Error != nil{
				err = result.Error
				resp.SetError(err.Error())
				log.Printf("[%08X] using storage fail: %s", id, err.Error())
				return executor.Sender.SendMessage(resp, request.GetSender())
			}
			//storage ready
			storageURL = result.Path
		}
		{
			var respChan = make(chan error, 1)
			executor.InstanceModule.UsingStorage(storageName, storageURL, respChan)
			err = <- respChan
			if err != nil{
				resp.SetError(err.Error())
				log.Printf("[%08X] update storage URL of instance to '%s' fail: %s", id, storageURL, err.Error())
				return executor.Sender.SendMessage(resp, request.GetSender())
			}
		}

	}
	if "" != networkName{
		gateway, err := request.GetString(framework.ParamKeyGateway)
		if err != nil{
			return err
		}
		dns, err := request.GetStringArray(framework.ParamKeyServer)
		if err != nil{
			return err
		}
		var respChan = make(chan error, 1)
		executor.NetworkModule.UpdateDHCPService(gateway, dns, respChan)
		err = <- respChan
		if err != nil{
			resp.SetError(err.Error())
			log.Printf("[%08X] update DHCP service fail: %s", id, err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	var respChan = make(chan []service.GuestConfig)
	executor.InstanceModule.GetAllInstance(respChan)
	allConfig := <- respChan
	var count = uint(len(allConfig))


	resp.SetSuccess(true)
	resp.SetUInt(framework.ParamKeyCount, count)
	if 0 == count{
		log.Printf("[%08X] no instance configured", id)
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	var names, ids, users, groups, secrets, addresses, systems, createTime, internal, external, hardware []string
	var cores, options, enables, progress, status, monitors, memories, disks, diskCounts, cpuPriorities, ioLimits []uint64
	for _, config := range allConfig {
		names = append(names, config.Name)
		ids = append(ids, config.ID)
		users = append(users, config.User)
		groups = append(groups, config.Group)
		cores = append(cores, uint64(config.Cores))
		if config.AutoStart{
			options = append(options, 1)
		}else{
			options = append(options, 0)
		}
		if config.Created{
			enables = append(enables, 1)
			progress = append(progress, 0)
		}else{
			enables = append(enables, 0)
			progress = append(progress, uint64(config.Progress))
		}
		if config.Running{
			status = append(status, service.InstanceStatusRunning)
		}else{
			status = append(status, service.InstanceStatusStopped)
		}
		monitors = append(monitors, uint64(config.MonitorPort))
		secrets = append(secrets, config.MonitorSecret)
		memories = append(memories, uint64(config.Memory))
		var diskCount = len(config.Disks)
		diskCounts = append(diskCounts, uint64(diskCount))
		for _, diskSize := range config.Disks{
			disks = append(disks, diskSize)
		}
		addresses = append(addresses, config.NetworkAddress)
		systems = append(systems, config.Template.OperatingSystem)
		createTime = append(createTime, config.CreateTime)
		internal = append(internal, config.InternalAddress)
		external = append(external, config.ExternalAddress)
		hardware = append(hardware, config.HardwareAddress)
		cpuPriorities = append(cpuPriorities, uint64(config.CPUPriority))
		ioLimits = append(ioLimits, []uint64{config.ReadSpeed, config.WriteSpeed,
			config.ReadIOPS, config.WriteIOPS, config.ReceiveSpeed, config.SendSpeed}...)
	}
	resp.SetStringArray(framework.ParamKeyName, names)
	resp.SetStringArray(framework.ParamKeyInstance, ids)
	resp.SetStringArray(framework.ParamKeyUser, users)
	resp.SetStringArray(framework.ParamKeyGroup, groups)
	resp.SetStringArray(framework.ParamKeySecret, secrets)
	resp.SetStringArray(framework.ParamKeyAddress, addresses)
	resp.SetStringArray(framework.ParamKeySystem, systems)
	resp.SetStringArray(framework.ParamKeyCreate, createTime)
	resp.SetStringArray(framework.ParamKeyInternal, internal)
	resp.SetStringArray(framework.ParamKeyExternal, external)
	resp.SetStringArray(framework.ParamKeyHardware, hardware)
	resp.SetUIntArray(framework.ParamKeyCore, cores)
	resp.SetUIntArray(framework.ParamKeyOption, options)
	resp.SetUIntArray(framework.ParamKeyEnable, enables)
	resp.SetUIntArray(framework.ParamKeyProgress, progress)
	resp.SetUIntArray(framework.ParamKeyStatus, status)
	resp.SetUIntArray(framework.ParamKeyMonitor, monitors)
	resp.SetUIntArray(framework.ParamKeyMemory, memories)
	resp.SetUIntArray(framework.ParamKeyCount, diskCounts)
	resp.SetUIntArray(framework.ParamKeyDisk, disks)
	resp.SetUIntArray(framework.ParamKeyPriority, cpuPriorities)
	resp.SetUIntArray(framework.ParamKeyLimit, ioLimits)
	log.Printf("[%08X] %d instance config(s) reported", id, count)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
