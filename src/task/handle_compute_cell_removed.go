package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
)

type HandleComputeCellRemovedExecutor struct {
	Sender         framework.MessageSender
	InstanceModule service.InstanceModule
	StorageModule  service.StorageModule
}

func (executor *HandleComputeCellRemovedExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	log.Printf("[%08X] recv cell removed from %s", id, request.GetSender())
	var respChan = make(chan error, 1)
	{
		//detach instance module
		executor.InstanceModule.DetachStorage(respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] detach instance module fail: %s", id, err.Error())
			return nil
		}
		log.Printf("[%08X] instance module detached", id)
	}
	{
		//detach storage module
		executor.StorageModule.DetachStorage(respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] detach storage module fail: %s", id, err.Error())
			return nil
		}
		log.Printf("[%08X] storage module detached", id)
	}
	return nil
}