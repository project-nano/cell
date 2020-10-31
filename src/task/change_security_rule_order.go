package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type ChangeSecurityRuleOrderExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ChangeSecurityRuleOrderExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	var direction, index int
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil{
		err = fmt.Errorf("get instance id fail: %s", err.Error())
		return
	}
	if index, err = request.GetInt(framework.ParamKeyIndex); err != nil{
		err = fmt.Errorf("get index fail: %s", err.Error())
		return
	}
	if direction, err = request.GetInt(framework.ParamKeyMode); err != nil{
		err = fmt.Errorf("get direction fail: %s", err.Error())
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.ChangeGuestRuleOrderResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	var moveUp = false
	if direction >= 0 {
		moveUp = true
		executor.InstanceModule.PullUpSecurityPolicyRule(instanceID, index, respChan)
	}else{
		executor.InstanceModule.PushDownSecurityPolicyRule(instanceID, index, respChan)
	}

	err = <- respChan
	if nil != err{
		log.Printf("[%08X] change order of %dth security rule of instance '%s' fail: %s",
			id, index, instanceID, err.Error())
		resp.SetError(err.Error())
	}else{
		if moveUp{
			log.Printf("[%08X] %dth security rule of instance '%s' moved up",
				id, index, instanceID)
		}else{
			log.Printf("[%08X] %dth security rule of instance '%s' moved down",
				id, index, instanceID)
		}
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
