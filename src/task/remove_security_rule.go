package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type RemoveSecurityRuleExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *RemoveSecurityRuleExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	var index int
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil{
		err = fmt.Errorf("get instance id fail: %s", err.Error())
		return
	}
	if index, err = request.GetInt(framework.ParamKeyIndex); err != nil{
		err = fmt.Errorf("get rule index fail: %s", err.Error())
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.RemoveGuestRuleResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.InstanceModule.RemoveSecurityPolicyRule(instanceID, index, respChan)
	err = <- respChan
	if nil != err{
		log.Printf("[%08X] remove %dth security rule of instance '%s' fail: %s",
			id, index, instanceID, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] %dth security rule of instance '%s' removed",
			id, index, instanceID)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}