package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
)

type DetachInstanceExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
	StorageModule   service.StorageModule
	NetworkModule   service.NetworkModule
}

func (executor *DetachInstanceExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	resp, _ := framework.CreateJsonMessage(framework.DetachInstanceResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	idList, err := request.GetStringArray(framework.ParamKeyInstance)
	if err != nil {
		log.Printf("[%08X] recv detach instance request from %s.[%08X] but get target intance fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var count = len(idList)
	if 0 == count{
		log.Printf("[%08X] recv purge all instances request from %s.[%08X]", id, request.GetSender(), request.GetFromSession())
	}else{
		log.Printf("[%08X] recv detach %d instance(s) request from %s.[%08X]", id, count, request.GetSender(), request.GetFromSession())
	}

	var respChan = make(chan error, 1)
	executor.NetworkModule.DetachInstances(idList, respChan)
	err = <-respChan
	if err != nil {
		resp.SetError(err.Error())
		log.Printf("[%08X] detach network resource fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	executor.StorageModule.DetachVolumeGroup(idList, respChan)
	err = <-respChan
	if err != nil {
		resp.SetError(err.Error())
		log.Printf("[%08X] detach storage volumes fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	executor.InstanceModule.DetachInstances(idList, respChan)
	err = <-respChan
	if err != nil {
		resp.SetError(err.Error())
		log.Printf("[%08X] detach instances fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	if 0 == count{
		log.Printf("[%08X] all instance(s) purgeed", id)
	}else{
		log.Printf("[%08X] %d instance(s) detached", id, count)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}