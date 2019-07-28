package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
	"fmt"
	"time"
	"github.com/pkg/errors"
)

type DeleteSnapshotExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
	StorageModule  service.StorageModule
}

func (executor *DeleteSnapshotExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	var snapshot string
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil{
		return err
	}
	if snapshot, err = request.GetString(framework.ParamKeyName); err != nil{
		return err
	}

	log.Printf("[%08X] recv delete snapshot '%s' of guest '%s' from %s.[%08X]",
		id, snapshot, instanceID, request.GetSender(), request.GetFromSession())
	resp, _ := framework.CreateJsonMessage(framework.DeleteSnapshotResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	{
		var respChan = make(chan service.InstanceResult, 1)
		executor.InstanceModule.GetInstanceStatus(instanceID, respChan)
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("[%08X] get instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}

		err = func(instance service.InstanceStatus) (err error){
			if !instance.Created{
				err = fmt.Errorf("instance '%s' not created", instanceID)
				return
			}
			//todo: allow operating on branch snapshots
			if instance.Running{
				err = errors.New("live snapshot not supported yes, shutdown instance first")
				return
			}
			return nil
		}(result.Instance)
		if err != nil{
			log.Printf("[%08X] check instance fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	{
		var respChan = make(chan error, 1)
		executor.StorageModule.DeleteSnapshot(instanceID, snapshot, respChan)
		var timer = time.NewTimer(service.DefaultOperateTimeout)
		select{
		case <- timer.C:
			err = errors.New("request timeout")
			log.Printf("[%08X] delete snapshot timeout", id)
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		case err = <-respChan:
			if err != nil{
				log.Printf("[%08X] delete snapshot fail: %s", id, err.Error())
				resp.SetError(err.Error())
			}else{
				log.Printf("[%08X] snapshot '%s' deleted from guest '%s'", id, snapshot, instanceID)
				resp.SetSuccess(true)
			}
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
