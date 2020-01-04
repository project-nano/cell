package task

import (
	"log"
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"fmt"
)

type ModifyNetworkThresholdExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ModifyNetworkThresholdExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	guestID, err := request.GetString(framework.ParamKeyGuest)
	if err != nil {
		return err
	}
	limitParameters, err := request.GetUIntArray(framework.ParamKeyLimit)
	if err != nil {
		return err
	}
	const (
		ReceiveOffset             = iota
		SendOffset
		ValidLimitParametersCount = 2
	)

	if ValidLimitParametersCount != len(limitParameters){
		var err = fmt.Errorf("invalid QoS parameters count %d", len(limitParameters))
		return err
	}
	var receiveSpeed = limitParameters[ReceiveOffset]
	var sendSpeed = limitParameters[SendOffset]

	log.Printf("[%08X] request modifying network threshold of guest '%s' from %s.[%08X]", id, guestID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyNetworkThresholdResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.InstanceModule.ModifyNetworkThreshold(guestID, receiveSpeed, sendSpeed, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] modify network threshold fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] network threshold of guest '%s' changed to receive %d Kps, send %d Kps", id, guestID,
			receiveSpeed >> 10, sendSpeed >> 10)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
