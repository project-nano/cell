package task

import (
	"errors"
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type ShrinkGuestVolumeExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
	StorageModule  service.StorageModule
}

func (executor *ShrinkGuestVolumeExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID string
	var index uint
	if guestID, err = request.GetString(framework.ParamKeyGuest); err != nil {
		return err
	}
	if index, err = request.GetUInt(framework.ParamKeyDisk); err != nil {
		return err
	}
	log.Printf("[%08X] recv shrink disk of guest '%s' from %s.[%08X]",
		id, guestID, request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ResizeDiskResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	var targetVolume string
	{
		var respChan = make(chan service.InstanceResult)
		executor.InstanceModule.GetInstanceStatus(guestID, respChan)
		var result = <-respChan
		if result.Error != nil {
			err = result.Error
			log.Printf("[%08X] get instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}

		err = func(instance service.InstanceStatus, index int) (err error) {
			if !instance.Created {
				err = fmt.Errorf("instance '%s' not created", guestID)
				return
			}
			if instance.Running {
				err = fmt.Errorf("instance '%s' not stopped", guestID)
				return
			}
			var volumeCount = len(instance.StorageVolumes)
			if 0 == volumeCount {
				err = errors.New("no volume available")
				return
			}
			if index >= volumeCount {
				err = fmt.Errorf("invalid disk index %d", index)
				return
			}
			return nil
		}(result.Instance, int(index))
		if err != nil {
			log.Printf("[%08X] check instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		targetVolume = result.Instance.StorageVolumes[int(index)]
	}
	var resultChan = make(chan service.StorageResult, 1)
	{
		executor.StorageModule.ShrinkVolume(id, guestID, targetVolume, resultChan)
		var timer = time.NewTimer(service.GetConfigurator().GetOperateTimeout())
		select {
		case <-timer.C:
			err = errors.New("request timeout")
			log.Printf("[%08X] shrink disk timeout", id)
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		case result := <-resultChan:
			if result.Error != nil {
				err = result.Error
				log.Printf("[%08X] shrink disk fail: %s", id, err.Error())
				resp.SetError(err.Error())
			} else {
				log.Printf("[%08X] volume %s shrank successfully", id, targetVolume)
				resp.SetSuccess(true)
			}
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
