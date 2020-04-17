package main

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"log"
	"time"
)

type hostStatus struct {
	Cores           uint
	CpuUsage        float64
	Memory          uint64
	MemoryAvailable uint64
	Disk            uint64
	DiskAvailable   uint64
}

type ioSnapshot struct {
	Timestamp      time.Time
	DiskWrite      uint64
	DiskRead       uint64
	NetworkSend    uint64
	NetworkReceive uint64
}

type ioCounter struct {
	Timestamp     time.Time
	Duration      time.Duration
	BytesWritten  uint64
	BytesRead     uint64
	BytesSent     uint64
	BytesReceived uint64
	WriteSpeed    uint64
	ReadSpeed     uint64
	SendSpeed     uint64
	ReceiveSpeed  uint64
}

type collectorCmd struct {
	Command collectorCommandType
	Name    string
}

type collectorCommandType int

const (
	collectCommandAdd = iota
	collectCommandRemove
)

type CollectorModule struct {
	sender                framework.MessageSender
	commands              chan collectorCmd
	instanceEvents        chan service.InstanceStatusChangedEvent
	onStoragePathsChanged chan []string
	localStoragePaths     []string
	runner                *framework.SimpleRunner
}

func CreateCollectorModule(sender framework.MessageSender,
	eventChan chan service.InstanceStatusChangedEvent, storageChan chan []string) (*CollectorModule, error) {
	const (
		DefaultQueueSize = 1 << 10
	)
	var module = CollectorModule{}
	module.sender = sender
	module.commands = make(chan collectorCmd, DefaultQueueSize)
	module.instanceEvents = eventChan
	module.onStoragePathsChanged = storageChan
	module.runner = framework.CreateSimpleRunner(module.Routine)
	return &module, nil
}

func (collector *CollectorModule) AddObserver(name string) error {
	collector.commands <- collectorCmd{collectCommandAdd, name}
	return nil
}

func (collector *CollectorModule) RemoveObserver(name string) error {
	collector.commands <- collectorCmd{collectCommandRemove, name}
	return nil
}

func (collector *CollectorModule) Start() error {
	return collector.runner.Start()
}

func (collector *CollectorModule) Stop() error {
	return collector.runner.Stop()
}

func (collector *CollectorModule) Routine(c framework.RoutineController) {
	const (
		reportInterval  = 2 * time.Second
		collectInterval = reportInterval
	)
	log.Println("<collector> module started")
	var observerMap = map[string]bool{}
	var latestIOSnapshot ioSnapshot
	var latestSnapshotAvailable = false
	var reportAvailable = false
	var reportMessage framework.Message
	var reportTicker = time.NewTicker(reportInterval)
	var collectTicker = time.NewTicker(collectInterval)

	//prepare cpu percentage
	cpu.Percent(0, false)

	for !c.IsStopping() {
		select {
		case <-reportTicker.C:
			//on report
			if !reportAvailable {
				break
			}
			if 0 == len(observerMap) {
				//no observer available
				break
			}
			for target, _ := range observerMap {
				if err := collector.sender.SendMessage(reportMessage, target); err != nil {
					log.Printf("<collector> warning: send report to %s fail: %s", target, err.Error())
				}
			}

		case <-collectTicker.C:
			//on collect
			status, err := collector.collectHostStatus()
			if err != nil {
				log.Printf("<collector> collect host status fail: %s", err.Error())
				break
			}
			if !latestSnapshotAvailable {
				//collect latest counter
				latestIOSnapshot, err = captureIOSnapshot()
				if err != nil {
					log.Printf("<collector> capture first io snapshot fail: %s", err.Error())
					break
				}
				latestSnapshotAvailable = true
				break
			}
			currentSnapshot, err := captureIOSnapshot()
			if err != nil {
				log.Printf("<collector> capture io snapshot fail: %s", err.Error())
				break
			}
			counter := computeIOCounter(latestIOSnapshot, currentSnapshot)
			latestIOSnapshot = currentSnapshot
			reportMessage, err = buildObserverNotifyMessage(status, counter)
			if err != nil {
				log.Printf("<collector> marshal report message fail: %s", err.Error())
				break
			}
			reportAvailable = true
		case <-c.GetNotifyChannel():
			//exit
			c.SetStopping()
		case cmd := <-collector.commands:
			switch cmd.Command {
			case collectCommandAdd:
				var coreName = cmd.Name
				if _, exists := observerMap[coreName]; exists {
					log.Printf("<collector> observer %s already exists", coreName)
					break
				}
				observerMap[coreName] = true
				log.Printf("<collector> new observer %s added", coreName)
			case collectCommandRemove:
				if _, exists := observerMap[cmd.Name]; !exists {
					log.Printf("<collector> invalid observer %s", cmd.Name)
				} else {
					delete(observerMap, cmd.Name)
					log.Printf("<collector> observer %s removed", cmd.Name)
				}
			default:
				log.Printf("<collector> invalid collector command %d", cmd.Command)
			}
		case paths := <- collector.onStoragePathsChanged:
			collector.localStoragePaths = paths
			log.Printf("<collector> local storage paths changed to %s", paths)
		case event := <-collector.instanceEvents:
			var msg framework.Message
			switch event.Event {
			case service.InstanceStarted:
				msg, _ = framework.CreateJsonMessage(framework.GuestStartedEvent)
				msg.SetFromSession(0)
				msg.SetString(framework.ParamKeyInstance, event.ID)
			case service.InstanceStopped:
				msg, _ = framework.CreateJsonMessage(framework.GuestStoppedEvent)
				msg.SetFromSession(0)
				msg.SetString(framework.ParamKeyInstance, event.ID)
			case service.AddressChanged:
				msg, _ = framework.CreateJsonMessage(framework.AddressChangedEvent)
				msg.SetFromSession(0)
				msg.SetString(framework.ParamKeyInstance, event.ID)
				msg.SetString(framework.ParamKeyAddress, event.Address)
			default:
				log.Printf("<collector> ignore invalid instance event type %d", event.Event)
			}
			collector.broadCastMessage(msg, observerMap)
		}
	}

	log.Println("<collector> module stopped")
	c.NotifyExit()
}

func (collector *CollectorModule) broadCastMessage(message framework.Message, observers map[string]bool) {
	if 0 == len(observers) {
		log.Println("<collector> ignore broadcast, cause no observer available")
		return
	}
	for receiver, _ := range observers {
		if err := collector.sender.SendMessage(message, receiver); err != nil {
			log.Printf("<collector> warnning: notify message %08X to %s fail: %s", message.GetID(), receiver, err.Error())
		}
	}
}

func (collector *CollectorModule)collectHostStatus() (hostStatus, error) {
	var status hostStatus
	count, err := cpu.Counts(true)
	if err != nil {
		return status, err
	}
	status.Cores = uint(count)
	usages, err := cpu.Percent(0, false)
	if err != nil {
		return status, err
	}
	if 1 != len(usages) {
		return status, fmt.Errorf("unpected cpu usages size %d", len(usages))
	}
	status.CpuUsage = usages[0]
	vm, err := mem.VirtualMemory()
	if err != nil {
		return status, err
	}
	status.Memory = vm.Total
	status.MemoryAvailable = vm.Available
	for _, path := range collector.localStoragePaths {
		usage, err := disk.Usage(path)
		if err != nil {
			return status, err
		}
		status.Disk += usage.Total
		status.DiskAvailable += usage.Free
	}
	return status, nil
}

func captureIOSnapshot() (ioSnapshot, error) {
	var snapshot ioSnapshot
	partitions, err := disk.Partitions(false)
	if err != nil {
		return snapshot, err
	}
	for _, partitionStat := range partitions {
		counters, err := disk.IOCounters(partitionStat.Device)
		if err != nil {
			return snapshot, err
		}
		for _, counter := range counters {
			//for devName, counter := range counters{
			snapshot.DiskWrite += counter.WriteBytes
			snapshot.DiskRead += counter.ReadBytes
			//log.Printf("disk io %s > %s: %d / %d", partitionStat.Device, devName, counter.WriteBytes, counter.ReadBytes)
		}
	}

	netStats, err := net.IOCounters(false)
	if err != nil {
		return snapshot, err
	}
	for _, stat := range netStats {
		snapshot.NetworkReceive += stat.BytesRecv
		snapshot.NetworkSend += stat.BytesSent
		//log.Printf("interface %s: %d / %d", stat.Name, stat.BytesSent, stat.BytesRecv)
	}
	//log.Printf("debug: snapshot disk %d / %d, network %d / %d", snapshot.DiskWrite, snapshot.DiskRead, snapshot.NetworkSend, snapshot.NetworkReceive)
	return snapshot, nil
}

func computeIOCounter(previous, current ioSnapshot) ioCounter {
	elapsed := current.Timestamp.Sub(previous.Timestamp)
	if elapsed < time.Second*1 {
		return ioCounter{current.Timestamp, elapsed, 0, 0, 0, 0,
			0, 0, 0, 0}
	}
	elapsedMilliSeconds := uint64(elapsed / time.Millisecond)
	var result = ioCounter{Timestamp: current.Timestamp, Duration: elapsed}
	result.BytesRead = current.DiskRead - previous.DiskRead
	result.BytesWritten = current.DiskWrite - previous.DiskWrite
	result.BytesSent = current.NetworkSend - previous.NetworkSend
	result.BytesReceived = current.NetworkReceive - previous.NetworkReceive
	result.WriteSpeed = result.BytesWritten * 1000 / elapsedMilliSeconds
	result.ReadSpeed = result.BytesRead * 1000 / elapsedMilliSeconds
	result.SendSpeed = result.BytesSent * 1000 / elapsedMilliSeconds
	result.ReceiveSpeed = result.BytesReceived * 1000 / elapsedMilliSeconds
	return result
}

func buildObserverNotifyMessage(status hostStatus, io ioCounter) (msg framework.Message, err error) {
	msg, err = framework.CreateJsonMessage(framework.CellStatusReportEvent)
	if err != nil {
		return msg, err
	}
	msg.SetUInt(framework.ParamKeyCore, status.Cores)
	msg.SetFloat(framework.ParamKeyUsage, status.CpuUsage)
	msg.SetUIntArray(framework.ParamKeyMemory, []uint64{status.MemoryAvailable, status.Memory})
	msg.SetUIntArray(framework.ParamKeyDisk, []uint64{status.DiskAvailable, status.Disk})
	msg.SetUIntArray(framework.ParamKeyIO, []uint64{io.BytesRead, io.BytesWritten, io.BytesReceived, io.BytesSent})
	msg.SetUIntArray(framework.ParamKeySpeed, []uint64{io.ReadSpeed, io.WriteSpeed, io.ReceiveSpeed, io.SendSpeed})
	return msg, nil
}
