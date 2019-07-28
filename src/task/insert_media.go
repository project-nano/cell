package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
)

type InsertMediaCoreExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *InsertMediaCoreExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID, mediaSource, host string
	var port uint
	instanceID, err = request.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	if mediaSource, err = request.GetString(framework.ParamKeyMedia); err != nil{
		return err
	}
	if host, err = request.GetString(framework.ParamKeyHost); err != nil{
		return err
	}
	if port, err = request.GetUInt(framework.ParamKeyPort); err != nil{
		return err
	}

	log.Printf("[%08X] request insert media '%s' into '%s' from %s.[%08X]", id, mediaSource, instanceID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.InsertMediaResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)

	var respChan = make(chan error, 1)
	var media = service.InstanceMediaConfig{Mode: service.MediaModeHTTPS, ID:mediaSource, Host:host, Port:port}
	executor.InstanceModule.AttachMedia(instanceID, media, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] insert media fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] instance media inserted", id)
		resp.SetSuccess(true)
		{
			//notify event
			event, _ := framework.CreateJsonMessage(framework.MediaAttachedEvent)
			event.SetFromSession(id)
			event.SetString(framework.ParamKeyInstance, instanceID)
			event.SetString(framework.ParamKeyMedia, mediaSource)
			executor.Sender.SendMessage(event, request.GetSender())
		}
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
