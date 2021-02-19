package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type ModifyAutoStartExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ModifyAutoStartExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID string
	var enable bool
	if guestID, err = request.GetString(framework.ParamKeyGuest); err != nil {
		err = fmt.Errorf("get guest id fail: %s", err.Error())
		return
	}
	if enable, err = request.GetBoolean(framework.ParamKeyEnable); err != nil{
		err = fmt.Errorf("get enable flag fail: %s", err.Error())
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.ModifyAutoStartResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.InstanceModule.ModifyAutoStart(guestID, enable, respChan)
	if err = <- respChan; err != nil{
		log.Printf("[%08X] modify auto start fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		if enable{
			log.Printf("[%08X] auto start of guest '%s' enabled", id, guestID)
		}else{
			log.Printf("[%08X] auto start of guest '%s' disabled", id, guestID)
		}
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
