package task

import (
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
	"math/rand"
)

type ModifyGuestPasswordExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
	RandomGenerator *rand.Rand
}

func (executor *ModifyGuestPasswordExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	const (
		PasswordLength = 10
	)
	var guestID, user, password string

	if guestID, err = request.GetString(framework.ParamKeyGuest);err != nil{
		return err
	}
	if user, err = request.GetString(framework.ParamKeyUser);err != nil{
		return err
	}
	if password, err = request.GetString(framework.ParamKeySecret);err != nil{
		return err
	}

	if "" == password{
		password = executor.generatePassword(PasswordLength)
		log.Printf("[%08X] new password '%s' generated for modify auth", id, password)
	}

	var respChan = make(chan service.InstanceResult)
	executor.InstanceModule.ModifyGuestAuth(guestID, password, user, respChan)

	resp, _ := framework.CreateJsonMessage(framework.ModifyAuthResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	result := <- respChan
	if result.Error != nil{
		resp.SetError(result.Error.Error())
		log.Printf("[%08X] modify password fail: %s", id, result.Error.Error())
	}else{
		resp.SetSuccess(true)
		resp.SetString(framework.ParamKeyUser, result.User)
		resp.SetString(framework.ParamKeySecret, result.Password)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}

func (executor *ModifyGuestPasswordExecutor)generatePassword(length int) (string){
	const (
		Letters = "~!@#$%^&*()_[]-=+0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	)
	var result = make([]byte, length)
	var n = len(Letters)
	for i := 0 ; i < length; i++{
		result[i] = Letters[executor.RandomGenerator.Intn(n)]
	}
	return string(result)
}
