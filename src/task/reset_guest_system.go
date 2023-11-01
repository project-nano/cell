package task

import (
	"errors"
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type ResetGuestSystemExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
	StorageModule  service.StorageModule
}

func (executor *ResetGuestSystemExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID, imageID, mediaHost string
	var mediaPort, imageSize uint
	if guestID, err = request.GetString(framework.ParamKeyGuest); err != nil {
		return
	}
	if imageID, err = request.GetString(framework.ParamKeyImage); err != nil {
		return
	}
	if mediaHost, err = request.GetString(framework.ParamKeyHost); err != nil {
		return err
	}
	if mediaPort, err = request.GetUInt(framework.ParamKeyPort); err != nil {
		return err
	}
	if imageSize, err = request.GetUInt(framework.ParamKeySize); err != nil {
		return err
	}
	log.Printf("[%08X] recv reset system of guest '%s' to image '%s' from %s.[%08X]",
		id, guestID, imageID, request.GetSender(), request.GetFromSession())
	resp, _ := framework.CreateJsonMessage(framework.ResetSystemResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var systemVolume string
	var systemSize uint64
	{
		var respChan = make(chan service.InstanceResult, 1)
		//check instance
		executor.InstanceModule.GetInstanceStatus(guestID, respChan)
		var result = <-respChan
		if result.Error != nil {
			err = result.Error
			log.Printf("[%08X] get instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		var ins = result.Instance
		if ins.Running {
			err = fmt.Errorf("guest '%s' is still running", ins.Name)
			log.Printf("[%08X] check instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		if 0 == len(ins.StorageVolumes) {
			err = fmt.Errorf("no volumes available for guest '%s'", ins.Name)
			log.Printf("[%08X] check instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		systemVolume = ins.StorageVolumes[0]
		systemSize = ins.Disks[0]
	}
	{
		//write system volume
		var startChan = make(chan error, 1)
		var progressChan = make(chan uint, 1)
		var resultChan = make(chan service.StorageResult, 1)
		executor.StorageModule.ReadDiskImage(id, guestID, systemVolume, imageID, systemSize, uint64(imageSize), mediaHost, mediaPort,
			startChan, progressChan, resultChan)
		//wait start
		{
			var timer = time.NewTimer(service.GetConfigurator().GetOperateTimeout())
			select {
			case err = <-startChan:
				if err != nil {
					log.Printf("[%08X] start reset system image fail: %s", id, err.Error())
					resp.SetError(err.Error())
					return executor.Sender.SendMessage(resp, request.GetSender())
				} else {
					//started
					log.Printf("[%08X] reset system image started...", id)
					resp.SetSuccess(true)
					executor.Sender.SendMessage(resp, request.GetSender())
				}

			case <-timer.C:
				//wait start timeout
				err = errors.New("start reset system image timeout")
				resp.SetError(err.Error())
				return executor.Sender.SendMessage(resp, request.GetSender())
			}
		}
		//update progress&wait finish
		const (
			CheckInterval = 2 * time.Second
		)

		resetEvent, _ := framework.CreateJsonMessage(framework.SystemResetEvent)
		resetEvent.SetFromSession(id)
		resetEvent.SetSuccess(false)
		resetEvent.SetString(framework.ParamKeyGuest, guestID)

		updateEvent, _ := framework.CreateJsonMessage(framework.GuestUpdatedEvent)
		updateEvent.SetFromSession(id)
		updateEvent.SetSuccess(true)
		updateEvent.SetString(framework.ParamKeyInstance, guestID)

		var latestUpdate = time.Now()
		var ticker = time.NewTicker(CheckInterval)
		for {
			select {
			case <-ticker.C:
				//check
				if time.Now().After(latestUpdate.Add(service.GetConfigurator().GetOperateTimeout())) {
					//timeout
					err = errors.New("wait reset progress timeout")
					log.Printf("[%08X] reset system image fail: %s", id, err.Error())
					resetEvent.SetError(err.Error())
					return executor.Sender.SendMessage(resetEvent, request.GetSender())
				}
			case progress := <-progressChan:
				latestUpdate = time.Now()
				updateEvent.SetUInt(framework.ParamKeyProgress, progress)
				log.Printf("[%08X] progress => %d %%", id, progress)
				if err = executor.Sender.SendMessage(updateEvent, request.GetSender()); err != nil {
					log.Printf("[%08X] warning: notify progress fail: %s", id, err.Error())
				}
			case result := <-resultChan:
				err = result.Error
				if err != nil {
					log.Printf("[%08X] reset system image fail: %s", id, err.Error())
					resetEvent.SetSuccess(false)
					resetEvent.SetError(err.Error())
					return executor.Sender.SendMessage(resetEvent, request.GetSender())
				}
				log.Printf("[%08X] reset system image success, %d MB in size", id, result.Size>>20)
				{
					var errChan = make(chan error, 1)
					executor.InstanceModule.ResetGuestSystem(guestID, errChan)
					if err = <-errChan; err != nil {
						log.Printf("[%08X] reset guest system fail: %s", id, err.Error())
						resetEvent.SetSuccess(false)
						resetEvent.SetError(err.Error())
						return executor.Sender.SendMessage(resetEvent, request.GetSender())
					}

				}
				//notify guest created
				resetEvent.SetSuccess(true)

				if err = executor.Sender.SendMessage(resetEvent, request.GetSender()); err != nil {
					log.Printf("[%08X] warning: notify instance created fail: %s", id, err.Error())
				}
				return nil
			}
		}

	}
	return nil
}
