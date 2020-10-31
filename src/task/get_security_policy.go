package task

import (
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type GetSecurityPolicyExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *GetSecurityPolicyExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil{
		err = fmt.Errorf("get instance id fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.GetGuestRuleResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var respChan = make(chan service.InstanceResult, 1)
	executor.InstanceModule.GetSecurityPolicy(instanceID, respChan)
	var result = <- respChan
	if nil != result.Error{
		err = result.Error
		log.Printf("[%08X] get security policy of instance '%s' fail: %s",
			id, instanceID, err.Error())
		resp.SetError(err.Error())
	}else{
		var policy = result.Policy
		var fromIP, toIP, toPort, protocols, actions []uint64
		for index, rule := range policy.Rules{
			fromIP = append(fromIP, uint64(service.IPv4ToUInt32(rule.SourceAddress)))
			toIP = append(toIP, uint64(service.IPv4ToUInt32(rule.TargetAddress)))
			toPort = append(toPort, uint64(rule.TargetPort))
			switch rule.Protocol {
			case service.PolicyRuleProtocolTCP:
				protocols = append(protocols, uint64(service.PolicyRuleProtocolIndexTCP))
			case service.PolicyRuleProtocolUDP:
				protocols = append(protocols, uint64(service.PolicyRuleProtocolIndexUDP))
			case service.PolicyRuleProtocolICMP:
				protocols = append(protocols, uint64(service.PolicyRuleProtocolIndexICMP))
			default:
				log.Printf("[%08X] warning: invalid protocol %s on %dth security rule of instance '%s'",
					id, rule.Protocol, index, instanceID)
				continue
			}
			if rule.Accept{
				actions = append(actions, service.PolicyRuleActionAccept)
			}else{
				actions = append(actions, service.PolicyRuleActionReject)
			}
		}
		if policy.Accept{
			actions = append(actions, service.PolicyRuleActionAccept)
			log.Printf("[%08X] %d security rule(s) available for instance '%s', accept by default",
				id, len(toPort), instanceID)
		}else{
			actions = append(actions, service.PolicyRuleActionReject)
			log.Printf("[%08X] %d security rule(s) available for instance '%s', reject by default",
				id, len(toPort), instanceID)
		}
		resp.SetUIntArray(framework.ParamKeyFrom, fromIP)
		resp.SetUIntArray(framework.ParamKeyTo, toIP)
		resp.SetUIntArray(framework.ParamKeyPort, toPort)
		resp.SetUIntArray(framework.ParamKeyProtocol, protocols)
		resp.SetUIntArray(framework.ParamKeyAction, actions)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
