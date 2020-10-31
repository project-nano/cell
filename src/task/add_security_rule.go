package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type AddSecurityRuleExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *AddSecurityRuleExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	var accept bool
	var fromIP, toIP, toPort, protocol uint
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil{
		err = fmt.Errorf("get instance id fail: %s", err.Error())
		return
	}
	if accept, err = request.GetBoolean(framework.ParamKeyAction); err != nil{
		err = fmt.Errorf("get action fail: %s", err.Error())
		return
	}
	if fromIP, err = request.GetUInt(framework.ParamKeyFrom); err != nil{
		err = fmt.Errorf("get source address fail: %s", err.Error())
		return
	}
	if toIP, err = request.GetUInt(framework.ParamKeyTo); err != nil{
		err = fmt.Errorf("get target address fail: %s", err.Error())
		return
	}
	if toPort, err = request.GetUInt(framework.ParamKeyPort); err != nil{
		err = fmt.Errorf("get target port fail: %s", err.Error())
		return
	}else if 0 == toPort || toPort > 0xFFFF{
		err = fmt.Errorf("invalid target port %d", toPort)
		return
	}
	if protocol, err = request.GetUInt(framework.ParamKeyProtocol); err != nil{
		err = fmt.Errorf("get protocol fail: %s", err.Error())
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.AddGuestRuleResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var rule = service.SecurityPolicyRule{
		Accept: accept,
		TargetPort: toPort,
	}

	switch protocol {
	case service.PolicyRuleProtocolIndexTCP:
		rule.Protocol = service.PolicyRuleProtocolTCP
	case service.PolicyRuleProtocolIndexUDP:
		rule.Protocol = service.PolicyRuleProtocolUDP
	case service.PolicyRuleProtocolIndexICMP:
		rule.Protocol = service.PolicyRuleProtocolICMP
	default:
		err = fmt.Errorf("invalid protocol %d for security rule", protocol)
		return
	}
	rule.SourceAddress = service.UInt32ToIPv4(uint32(fromIP))
	rule.TargetAddress = service.UInt32ToIPv4(uint32(toIP))

	var respChan = make(chan error, 1)
	executor.InstanceModule.AddSecurityPolicyRule(instanceID, rule, respChan)
	err = <- respChan
	if nil != err{
		log.Printf("[%08X] add security rule to instance '%s' fail: %s",
			id, instanceID, err.Error())
		resp.SetError(err.Error())
	}else{
		if accept{
			log.Printf("[%08X] add security rule to instance '%s': accept protocol '%s' from '%s' to '%s:%d'",
				id, instanceID, rule.Protocol, rule.SourceAddress, rule.TargetAddress, rule.TargetPort)
		}else{
			log.Printf("[%08X] add security rule to instance '%s': reject protocol '%s' from '%s' to '%s:%d'",
				id, instanceID, rule.Protocol, rule.SourceAddress, rule.TargetAddress, rule.TargetPort)
		}
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
