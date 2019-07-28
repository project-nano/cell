package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
	"time"
	"errors"
)

type QuerySnapshotExecutor struct {
	Sender         framework.MessageSender
	StorageModule  service.StorageModule
}

func (executor *QuerySnapshotExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID string
	if instanceID, err = request.GetString(framework.ParamKeyInstance); err != nil{
		return err
	}

	log.Printf("[%08X] recv query snapshots for guest '%s' from %s.[%08X]",
		id, instanceID, request.GetSender(), request.GetFromSession())
	resp, _ := framework.CreateJsonMessage(framework.QuerySnapshotResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	{
		var respChan = make(chan service.StorageResult, 1)
		executor.StorageModule.QuerySnapshot(instanceID, respChan)
		var timer = time.NewTimer(service.DefaultOperateTimeout)
		select{
		case <- timer.C:
			err = errors.New("request timeout")
			log.Printf("[%08X] query snapshot timeout", id)
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		case result := <- respChan:
			if result.Error != nil{
				err = result.Error
				log.Printf("[%08X] query snapshot fail: %s", id, err.Error())
				resp.SetError(err.Error())
			}else{
				var snapshotList = result.SnapshotList
				var names, backings []string
				var rootFlags, currentFlags []uint64
				for _, snapshot := range snapshotList{
					names = append(names, snapshot.Name)
					backings = append(backings, snapshot.Backing)
					if snapshot.IsRoot{
						rootFlags = append(rootFlags, 1)
					}else{
						rootFlags = append(rootFlags, 0)
					}
					if snapshot.IsCurrent{
						currentFlags = append(currentFlags, 1)
					}else{
						currentFlags = append(currentFlags, 0)
					}
				}
				resp.SetStringArray(framework.ParamKeyName, names)
				resp.SetStringArray(framework.ParamKeyPrevious, backings)
				resp.SetUIntArray(framework.ParamKeySource, rootFlags)
				resp.SetUIntArray(framework.ParamKeyCurrent, currentFlags)
				log.Printf("[%08X] %d snapshot(s) available for guest '%s'", id, len(snapshotList), instanceID)
				resp.SetSuccess(true)
			}
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}

