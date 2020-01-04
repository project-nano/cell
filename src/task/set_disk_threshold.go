package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"log"
	"fmt"
)

type ModifyDiskThresholdExecutor struct {
	Sender         framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ModifyDiskThresholdExecutor)Execute(id framework.SessionID, request framework.Message,
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
		ReadSpeedOffset           = iota
		WriteSpeedOffset
		ReadIOPSOffset
		WriteIOPSOffset
		ValidLimitParametersCount = 4
	)

	if ValidLimitParametersCount != len(limitParameters){
		var err = fmt.Errorf("invalid QoS parameters count %d", len(limitParameters))
		return err
	}
	var readSpeed = limitParameters[ReadSpeedOffset]
	var writeSpeed = limitParameters[WriteSpeedOffset]
	var readIOPS = limitParameters[ReadIOPSOffset]
	var writeIOPS = limitParameters[WriteIOPSOffset]

	log.Printf("[%08X] request modifying disk threshold of guest '%s' from %s.[%08X]", id, guestID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyDiskThresholdResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.InstanceModule.ModifyDiskThreshold(guestID, readSpeed, readIOPS, writeSpeed, writeIOPS, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] modify disk threshold fail: %s", id, err.Error())
		resp.SetError(err.Error())
	}else{
		//log.Printf("[%08X] disk threshold of guest '%s' changed to read (%d MB/s, %d ops), write (%d MB/s, %d ops)", id, guestID,
		//	readSpeed >> 20, readIOPS, writeSpeed >> 20, writeIOPS)
		log.Printf("[%08X] disk threshold of guest '%s' changed to read %d, write %d per second", id, guestID,
			readIOPS, writeIOPS)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
