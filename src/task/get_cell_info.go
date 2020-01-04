package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"log"
)

type GetCellInfoExecutor struct {
	Sender          framework.MessageSender
	InstanceModule  service.InstanceModule
	StorageModule   service.StorageModule
	NetworkModule   service.NetworkModule

}

func (executor *GetCellInfoExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {

	//todo: add instance/network info
	resp, _ := framework.CreateJsonMessage(framework.GetComputePoolCellResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)

	{
		//storage
		var respChan = make(chan service.StorageResult, 1)
		executor.StorageModule.GetAttachDevices(respChan)
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("[%08X] fetch attach device fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		var names, errMessages []string
		var attached []uint64
		for _, device := range result.Devices{
			names = append(names, device.Name)
			errMessages = append(errMessages, device.Error)
			if device.Attached{
				attached = append(attached, 1)
			}else{
				attached = append(attached, 0)
			}
		}
		resp.SetStringArray(framework.ParamKeyStorage, names)
		resp.SetStringArray(framework.ParamKeyError, errMessages)
		resp.SetUIntArray(framework.ParamKeyAttach, attached)
		log.Printf("[%08X] %d device(s) available", id, len(names))
	}
	resp.SetSuccess(true)
	return executor.Sender.SendMessage(resp, request.GetSender())
}