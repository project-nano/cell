package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
	"fmt"
	"time"
	"errors"
)

type ResizeGuestVolumeExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
	StorageModule  service.StorageModule
}

func (executor *ResizeGuestVolumeExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID string
	var index, size uint
	if guestID, err = request.GetString(framework.ParamKeyGuest); err != nil{
		return err
	}
	if index, err = request.GetUInt(framework.ParamKeyDisk); err != nil{
		return err
	}
	if size, err = request.GetUInt(framework.ParamKeySize); err != nil{
		return err
	}
	log.Printf("[%08X] recv resize disk of guest '%s' from %s.[%08X]",
		id, guestID, request.GetSender(), request.GetFromSession())
	resp, _ := framework.CreateJsonMessage(framework.ResizeDiskResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	var targetVolume string
	var targetSize = uint64(size)
	var targetIndex = int(index)
	{
		var respChan = make(chan service.InstanceResult)
		executor.InstanceModule.GetInstanceStatus(guestID, respChan)
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("[%08X] get instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}

		err = func(instance service.InstanceStatus, index int, size uint64) (err error){
			if !instance.Created{
				err = fmt.Errorf("instance '%s' not created", guestID)
				return
			}
			if instance.Running{
				err = fmt.Errorf("instance '%s' not stopped", guestID)
				return
			}
			var volumeCount = len(instance.StorageVolumes)
			if 0 == volumeCount{
				err = errors.New("no volume available")
				return
			}
			if index >= volumeCount{
				err = fmt.Errorf("invalid disk index %d", index)
				return
			}
			if instance.Disks[index] >= size{
				err = fmt.Errorf("must larger than current volume size %d GB", instance.Disks[index] >> 30)
				return
			}
			return nil
		}(result.Instance, targetIndex, targetSize)
		if err != nil{
			log.Printf("[%08X] check instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		targetVolume = result.Instance.StorageVolumes[targetIndex]
	}
	var resultChan = make(chan service.StorageResult, 1)
	{
		executor.StorageModule.ResizeVolume(id, guestID, targetVolume, targetSize, resultChan)
		var timer = time.NewTimer(service.DefaultOperateTimeout)
		select{
		case <- timer.C:
			err = errors.New("request timeout")
			log.Printf("[%08X] resize disk timeout", id)
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		case result := <- resultChan:
			if result.Error != nil{
				err = result.Error
				log.Printf("[%08X] resize disk fail: %s", id, err.Error())
				resp.SetError(err.Error())
			}else{
				{
					var respChan = make(chan error)
					executor.InstanceModule.UpdateDiskSize(guestID, targetIndex, targetSize, respChan)
					err = <- respChan
					if err != nil{
						log.Printf("[%08X] update disk size fail: %s", id, err.Error())
						resp.SetError(err.Error())
						return executor.Sender.SendMessage(resp, request.GetSender())
					}
				}
				log.Printf("[%08X] volume %s changed to %d GiB", id, targetVolume, targetSize >> 30)
				resp.SetSuccess(true)
			}
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
