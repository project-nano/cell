package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type ChangeDefaultSecurityActionExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ChangeDefaultSecurityActionExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	var accept bool
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil{
		err = fmt.Errorf("get instance id fail: %s", err.Error())
		return
	}
	if accept, err = request.GetBoolean(framework.ParamKeyAction); err != nil{
		err = fmt.Errorf("get action fail: %s", err.Error())
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.ChangeGuestRuleDefaultActionResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.InstanceModule.ChangeDefaultSecurityPolicyAction(instanceID, accept, respChan)
	err = <- respChan
	if nil != err{
		log.Printf("[%08X] change default security policy action of instance '%s' fail: %s",
			id, instanceID, err.Error())
		resp.SetError(err.Error())
	}else{
		if accept{
			log.Printf("[%08X] default security policy action of instance '%s' changed to accept",
				id, instanceID)
		}else{
			log.Printf("[%08X] default security policy action of instance '%s' changed to drop",
				id, instanceID)
		}
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
