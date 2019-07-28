package task

import (
	"github.com/project-nano/framework"
	"log"
	"service"
)

type GetGuestPasswordExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
}

func (executor *GetGuestPasswordExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	guestID, err := request.GetString(framework.ParamKeyGuest)
	if err != nil{
		return err
	}

	//log.Printf("[%08X] request get password of '%s' from %s.[%08X]", id, guestID,
	//	request.GetSender(), request.GetFromSession())

	var respChan = make(chan service.InstanceResult)
	executor.InstanceModule.GetGuestAuth(guestID, respChan)

	resp, _ := framework.CreateJsonMessage(framework.GetAuthResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	result := <- respChan
	if result.Error != nil{
		resp.SetError(result.Error.Error())
		log.Printf("[%08X] get password fail: %s", id, result.Error.Error())
	}else{
		resp.SetSuccess(true)
		resp.SetString(framework.ParamKeyUser, result.User)
		resp.SetString(framework.ParamKeySecret, result.Password)

	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}