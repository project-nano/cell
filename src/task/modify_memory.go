package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
)

type ModifyGuestMemoryExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ModifyGuestMemoryExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	guestID, err := request.GetString(framework.ParamKeyGuest)
	if err != nil {
		return err
	}
	memory, err := request.GetUInt(framework.ParamKeyMemory)
	if err != nil {
		return err
	}
	log.Printf("[%08X] request modifying memory of '%s' from %s.[%08X]", id, guestID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyMemoryResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	var respChan = make(chan error)
	executor.InstanceModule.ModifyGuestMemory(guestID, memory, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] modify memory fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] memory of guest '%s' changed to %d MB", id, guestID, memory / (1 << 20))
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}

