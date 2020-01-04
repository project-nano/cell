package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"log"
)

type EjectMediaCoreExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *EjectMediaCoreExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := request.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	log.Printf("[%08X] request eject media from '%s' from %s.[%08X]", id, instanceID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.EjectMediaResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)

	var respChan = make(chan error, 1)
	executor.InstanceModule.DetachMedia(instanceID, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] eject media fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] instance media ejected", id)
		resp.SetSuccess(true)
		{
			//notify event
			event, _ := framework.CreateJsonMessage(framework.MediaDetachedEvent)
			event.SetFromSession(id)
			event.SetString(framework.ParamKeyInstance, instanceID)
			executor.Sender.SendMessage(event, request.GetSender())
		}
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
