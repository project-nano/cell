package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type StartInstanceExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
	StorageModule  service.StorageModule
}

func (executor *StartInstanceExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	const (
		InstanceMediaOptionNone uint = iota
		InstanceMediaOptionImage
		InstanceMediaOptionNetwork
	)

	resp, _ := framework.CreateJsonMessage(framework.StartInstanceResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	var notified = false
	defer func() {
		if !notified && err != nil {
			resp.SetError(err.Error())
			_ = executor.Sender.SendMessage(resp, request.GetSender())
		}
	}()

	var instanceID string
	instanceID, err = request.GetString(framework.ParamKeyInstance)
	if err != nil {
		return
	}
	var mediaOption uint
	mediaOption, err = request.GetUInt(framework.ParamKeyOption)
	if err != nil {
		return
	}
	var respChan = make(chan error, 1)
	executor.StorageModule.ValidateVolumesForStart(instanceID, respChan)
	err = <-respChan
	if err != nil {
		log.Printf("[%08X] recv start instance '%s' from %s.[%08X] but validate volumes fail: %s",
			id, instanceID, request.GetSender(), request.GetFromSession(), err.Error())
		return
	}

	var mediaSource string
	switch mediaOption {
	case InstanceMediaOptionNone:
		//no media attached
		log.Printf("[%08X] request start instance '%s' from %s.[%08X]",
			id, instanceID, request.GetSender(), request.GetFromSession())
		executor.InstanceModule.StartInstance(instanceID, respChan)
	case InstanceMediaOptionImage:
		var host string
		var port uint
		if host, err = request.GetString(framework.ParamKeyHost); err != nil {
			return
		}
		if mediaSource, err = request.GetString(framework.ParamKeySource); err != nil {
			return
		}
		if port, err = request.GetUInt(framework.ParamKeyPort); err != nil {
			return
		}
		var media = service.InstanceMediaConfig{Mode: service.MediaModeHTTPS, ID: mediaSource, Host: host, Port: port}
		log.Printf("[%08X] request start instance '%s' with media '%s' (host %s:%d) from %s.[%08X]",
			id, instanceID, mediaSource, host, port, request.GetSender(), request.GetFromSession())
		executor.InstanceModule.StartInstanceWithMedia(instanceID, media, respChan)
	default:
		return fmt.Errorf("unsupported media option %d", mediaOption)
	}

	err = <-respChan
	if err != nil {
		log.Printf("[%08X] start instance fail: %s", id, err.Error())
		return
	}
	resp.SetSuccess(true)
	log.Printf("[%08X] start instance success", id)
	notified = true
	if err = executor.Sender.SendMessage(resp, request.GetSender()); err != nil {
		log.Printf("[%08X] warning: send response fail: %s", id, err.Error())
		return err
	}
	//notify
	event, _ := framework.CreateJsonMessage(framework.GuestStartedEvent)
	event.SetFromSession(id)
	event.SetString(framework.ParamKeyInstance, instanceID)
	if err = executor.Sender.SendMessage(event, request.GetSender()); err != nil {
		log.Printf("[%08X] warning: notify instance started fail: %s", id, err.Error())
		return err
	}
	if InstanceMediaOptionImage == mediaOption {
		//notify media attached
		attached, _ := framework.CreateJsonMessage(framework.MediaAttachedEvent)
		attached.SetFromSession(id)
		attached.SetString(framework.ParamKeyInstance, instanceID)
		attached.SetString(framework.ParamKeyMedia, mediaSource)
		if err = executor.Sender.SendMessage(attached, request.GetSender()); err != nil {
			log.Printf("[%08X] warning: notify media attached fail: %s", id, err.Error())
			return err
		}
	}
	return nil
}
