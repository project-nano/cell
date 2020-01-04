package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"log"
)

type ModifyCPUPriorityExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ModifyCPUPriorityExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	guestID, err := request.GetString(framework.ParamKeyGuest)
	if err != nil {
		return err
	}
	priorityValue, err := request.GetUInt(framework.ParamKeyPriority)
	if err != nil {
		return err
	}
	log.Printf("[%08X] request changing CPU priority of guest '%s' to %d from %s.[%08X]", id, guestID,
		priorityValue, request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyPriorityResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.InstanceModule.ModifyCPUPriority(guestID, service.PriorityEnum(priorityValue), respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] modify CPU priority fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] CPU priority of guest '%s' changed to %d", id, guestID, priorityValue)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
