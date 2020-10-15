package task

import (
	"fmt"
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"log"
	"strings"
)

type HandleAddressPoolChangedExecutor struct {
	InstanceModule service.InstanceModule
	NetworkModule  service.NetworkModule
}

func (executor *HandleAddressPoolChangedExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var allocationMode, gateway string
	var dns []string
	if gateway, err = request.GetString(framework.ParamKeyGateway); err != nil{
		return err
	}
	if dns, err = request.GetStringArray(framework.ParamKeyServer); err != nil{
		return
	}
	if allocationMode, err = request.GetString(framework.ParamKeyMode); err != nil{
		err = fmt.Errorf("get allocation mode fail: %s", err.Error())
		return
	}
	switch allocationMode {
	case service.AddressAllocationNone:
	case service.AddressAllocationDHCP:
	case service.AddressAllocationCloudInit:
		break
	default:
		err = fmt.Errorf("invalid allocation mode :%s", allocationMode)
		return
	}
	var respChan = make(chan error, 1)
	executor.NetworkModule.UpdateAddressAllocation(gateway, dns, allocationMode, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] update address allocation fail when address pool changed from %s.[%08X]: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
	}else{
		log.Printf("[%08X] address allocation updated to mode %s, gateway: %s, DNS: %s",
			id, allocationMode, gateway, strings.Join(dns, "/"))
		if service.AddressAllocationNone != allocationMode{
			executor.InstanceModule.SyncAddressAllocation(allocationMode)
		}
	}
	return nil
}
