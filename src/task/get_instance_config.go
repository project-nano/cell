package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
)

type GetInstanceConfigExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
}

func (executor *GetInstanceConfigExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	instanceID, err = request.GetString(framework.ParamKeyInstance)
	if err != nil{
		return err
	}
	log.Printf("[%08X] request get config of instance '%s' from %s.[%08X]",
		id, instanceID, request.GetSender(), request.GetFromSession())
	var respChan = make(chan service.InstanceResult)
	executor.InstanceModule.GetInstanceConfig(instanceID, respChan)

	resp, _ := framework.CreateJsonMessage(framework.GetGuestResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	result := <- respChan
	if result.Error != nil{
		resp.SetSuccess(false)
		resp.SetError(result.Error.Error())
		log.Printf("[%08X] get instance status fail: %s", id, result.Error.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var c = result.Instance.GuestConfig
	resp.SetSuccess(true)
	c.Marshal(resp)

	log.Printf("[%08X] query instance config success", id)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
