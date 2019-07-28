package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
)

type ModifyGuestNameExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ModifyGuestNameExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	guestID, err := request.GetString(framework.ParamKeyGuest)
	if err != nil {
		return err
	}
	name, err := request.GetString(framework.ParamKeyName)
	if err != nil {
		return err
	}
	log.Printf("[%08X] request rename guest '%s' from %s.[%08X]", id, guestID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyGuestNameResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	var respChan = make(chan error)
	executor.InstanceModule.ModifyGuestName(guestID, name, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] rename guest fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] guest '%s' renamed to %s", id, guestID, name)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
