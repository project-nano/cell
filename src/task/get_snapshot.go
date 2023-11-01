package task

import (
	"errors"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type GetSnapshotExecutor struct {
	Sender        framework.MessageSender
	StorageModule service.StorageModule
}

func (executor *GetSnapshotExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID, snapshotName string
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil {
		return err
	}
	if snapshotName, err = request.GetString(framework.ParamKeyName); err != nil {
		return err
	}

	log.Printf("[%08X] recv get snapshot '%s' for guest '%s' from %s.[%08X]",
		id, snapshotName, instanceID, request.GetSender(), request.GetFromSession())
	resp, _ := framework.CreateJsonMessage(framework.GetSnapshotResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	{
		var respChan = make(chan service.StorageResult, 1)
		executor.StorageModule.GetSnapshot(instanceID, snapshotName, respChan)
		var timer = time.NewTimer(service.GetConfigurator().GetOperateTimeout())
		select {
		case <-timer.C:
			err = errors.New("request timeout")
			log.Printf("[%08X] get snapshot timeout", id)
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		case result := <-respChan:
			if result.Error != nil {
				err = result.Error
				log.Printf("[%08X] get snapshot fail: %s", id, err.Error())
				resp.SetError(err.Error())
			} else {
				var snapshot = result.Snapshot
				resp.SetBoolean(framework.ParamKeyStatus, snapshot.Running)
				resp.SetString(framework.ParamKeyDescription, snapshot.Description)
				resp.SetString(framework.ParamKeyCreate, snapshot.CreateTime)
				resp.SetSuccess(true)
			}
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
