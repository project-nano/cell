package task

import (
	"github.com/project-nano/framework"
	"log"
	"service"
)

type ModifyGuestCoreExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ModifyGuestCoreExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	guestID, err := request.GetString(framework.ParamKeyGuest)
	if err != nil {
		return err
	}
	cores, err := request.GetUInt(framework.ParamKeyCore)
	if err != nil {
		return err
	}
	log.Printf("[%08X] request modifying cores of '%s' from %s.[%08X]", id, guestID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyCoreResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	var respChan = make(chan error)
	executor.InstanceModule.ModifyGuestCore(guestID, cores, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] modify core fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] cores of guest '%s' changed to %d", id, guestID, cores)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
