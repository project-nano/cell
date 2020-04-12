package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type ChangeStoragePathExecutor struct {
	Sender  framework.MessageSender
	Storage service.StorageModule
}

func (executor *ChangeStoragePathExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var newPath string
	if newPath, err = request.GetString(framework.ParamKeyPath); err != nil{
		err = fmt.Errorf("get new path fail: %s", err.Error())
		return
	}
	var respChan = make(chan error, 1)
	executor.Storage.ChangeDefaultStoragePath(newPath, respChan)

	resp, _ := framework.CreateJsonMessage(framework.ModifyCellStorageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	err = <- respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] change storage path fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}else{
		resp.SetSuccess(true)
		log.Printf("[%08X] default storage path changed to: %s", id, newPath)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}