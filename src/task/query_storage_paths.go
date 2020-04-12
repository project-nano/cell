package task

import (
	"github.com/project-nano/cell/service"
	"github.com/project-nano/framework"
	"log"
)

type QueryStoragePathExecutor struct {
	Sender  framework.MessageSender
	Storage service.StorageModule
}

func (executor *QueryStoragePathExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var respChan = make(chan service.StorageResult, 1)
	executor.Storage.QueryStoragePaths(respChan)

	resp, _ := framework.CreateJsonMessage(framework.QueryCellStorageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] query storage paths fail: %s", id, err.Error())
	}else{
		//parse result
		resp.SetSuccess(true)
		resp.SetUInt(framework.ParamKeyMode, uint(result.StorageMode))
		resp.SetStringArray(framework.ParamKeySystem, result.SystemPaths)
		resp.SetStringArray(framework.ParamKeyData, result.DataPaths)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}