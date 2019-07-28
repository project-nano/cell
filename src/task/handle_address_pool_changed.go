package task

import (
	"github.com/project-nano/framework"
	"service"
	"log"
	"strings"
)

type HandleAddressPoolChangedExecutor struct {
	NetworkModule  service.NetworkModule
}

func (executor *HandleAddressPoolChangedExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	gateway, err := request.GetString(framework.ParamKeyGateway)
	if err != nil{
		return err
	}
	dns, err := request.GetStringArray(framework.ParamKeyServer)
	if err != nil{
		return err
	}
	var respChan = make(chan error, 1)
	executor.NetworkModule.UpdateDHCPService(gateway, dns, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] update DHCP service fail when address pool changed from %s.[%08X]: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
	}else{
		log.Printf("[%08X] DHCP updated to gateway: %s, DNS: %s", id, gateway, strings.Join(dns, "/"))
	}
	return nil
}
