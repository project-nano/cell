package task

import (
	"errors"
	"fmt"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type ResetMonitorSecretExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
}

func (executor *ResetMonitorSecretExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID string
	if guestID, err = request.GetString(framework.ParamKeyGuest);err != nil{
		err = fmt.Errorf("get guest id fail: %s", err.Error())
		return err
	}
	var respChan = make(chan service.InstanceResult)
	executor.InstanceModule.ResetMonitorPassword(guestID, respChan)

	resp, _ := framework.CreateJsonMessage(framework.ResetSecretResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var password string
	result := <- respChan
	if result.Error != nil{
		err = result.Error

	}else{
		password = result.Password
		if "" == password{
			err = errors.New("new password is empty")
		}
	}
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] reset monitor secret fail: %s", id, err.Error())
	}else{
		resp.SetSuccess(true)
		resp.SetString(framework.ParamKeySecret, password)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
