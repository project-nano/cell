package service

import (
	"net"
	"github.com/project-nano/framework"
	"github.com/krolaw/dhcp4"
	"log"
	"time"
	"fmt"
	"strings"
)

type clientLease struct {
	IP      string
	Netmask string
	Gateway string
	DNS     []string
	Expire  time.Time
	Options dhcp4.Options
}

type DHCPHandler struct {
	operates chan dhcpOperate
	serverIP net.IP
}

type operateType int

const (
	opAllocate = iota
	opUpdate
	opDeallocate
	opChange
)

type dhcpOperate struct {
	Type     operateType
	MAC      string
	IP       string
	Gateway  string
	DNS      []string
	RespChan chan operateResult
}

type operateResult struct {
	Error   error
	IP      net.IP
	Options dhcp4.Options
}

type DHCPService struct {
	dhcpConn      *net.UDPConn
	handler       *DHCPHandler
	operates      chan dhcpOperate
	leases        map[string]clientLease //key = HW address
	networkModule *NetworkManager
	runner        *framework.SimpleRunner
}

const (
	leaseTimeout    = 12 * time.Hour
	confirmDuration = 3 * time.Minute
	checkInterval   = 1 * time.Minute
)

func CreateDHCPService(netModule *NetworkManager) (service* DHCPService, err error) {
	const (
		DHCPAddress = ":67"
	)
	listenAddress, err := net.ResolveUDPAddr("udp", DHCPAddress)
	if err != nil{
		return
	}
	serverAddress, err := GetCurrentIPOfDefaultBridge()
	if err != nil{
		return
	}
	serverIP, err := stringToIPv4(serverAddress)
	if err != nil{
		return
	}
	service = &DHCPService{}
	service.operates = make(chan dhcpOperate, 1 << 10)
	service.networkModule = netModule
	service.leases = map[string]clientLease{}
	netModule.OnDHCPUpdated = service.updateServer
	service.handler = &DHCPHandler{service.operates, serverIP}
	service.dhcpConn, err = net.ListenUDP("udp", listenAddress)
	if err != nil{
		err = fmt.Errorf("listen on DHCP port fail, please disable dnsmasq or other DHCP service.\nmessage: %s", err.Error())
		return
	}
	service.runner = framework.CreateSimpleRunner(service.Routine)
	return
}

func (service *DHCPService) updateServer(gateway string, dns []string){
	service.operates <- dhcpOperate{Type:opChange, Gateway:gateway, DNS:dns}
}

func (service *DHCPService) Start() error{
	return service.runner.Start()
}

func (service *DHCPService) Stop() error{
	return service.runner.Stop()
}

func (service *DHCPService) Routine(c framework.RoutineController)  {
	log.Println("<dhcp> service started")
	go func() {
		var err = dhcp4.Serve(service.dhcpConn, service.handler)
		log.Printf("<dhcp> handler stopped: %s", err.Error())
	}()
	var checkTicker = time.NewTicker(checkInterval)
	for !c.IsStopping() {
		select {
		case <-c.GetNotifyChannel():
			c.SetStopping()
			service.dhcpConn.Close()
		case op := <-service.operates:
			service.handleOperate(op)
		case <-checkTicker.C:
			service.clearExpiredLeases()
		}
	}
	c.NotifyExit()
	log.Println("<dhcp> service stopped")
}

func (service *DHCPService) clearExpiredLeases(){
	var now = time.Now()
	var expired []string
	for macAddress, lease := range service.leases{
		if lease.Expire.Before(now){
			expired = append(expired, macAddress)
		}
	}
	for _, mac := range expired{
		log.Printf("<dhcp> release expired lease for MAC '%s'", mac)
		delete(service.leases, mac)
	}
}

func (service *DHCPService)handleOperate(op dhcpOperate){
	var err error
	switch op.Type {
	case opAllocate:
		var macAddress = op.MAC
		if _, exists := service.leases[macAddress]; exists{
			delete(service.leases, macAddress)
			log.Printf("<dhcp> warning: previuos lease for MAC '%s' released", macAddress)
		}
		var respChan = make(chan NetworkResult, 1)
		service.networkModule.GetAddressByHWAddress(macAddress, respChan)
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			op.RespChan <- operateResult{Error:err}
			return
		}
		var newLease = clientLease{}
		clientIP, clientNet, err := net.ParseCIDR(result.Internal)
		if err != nil{
			err = result.Error
			op.RespChan <- operateResult{Error:err}
			return
		}
		newLease.IP = clientIP.String()
		newLease.Gateway = result.Gateway
		newLease.DNS = result.DNS
		//must confirm in duration
		newLease.Expire = time.Now().Add(confirmDuration)
		var netMask = clientNet.Mask
		newLease.Netmask = net.IPv4(netMask[0], netMask[1], netMask[2], netMask[3]).String()
		gatewayIP, err := stringToIPv4(result.Gateway)
		if err != nil{
			err = fmt.Errorf("parse gateway fail: %s", err.Error())
			op.RespChan <- operateResult{Error:err}
			return
		}
		var dnsBytes []byte
		for _, dns := range result.DNS{
			ip, err := stringToIPv4(dns)
			if err != nil{
				err = fmt.Errorf("parse DNS fail: %s", err.Error())
				op.RespChan <- operateResult{Error:err}
				return
			}
			dnsBytes = append(dnsBytes, ip...)
		}
		newLease.Options = dhcp4.Options{
			dhcp4.OptionSubnetMask: netMask,
			dhcp4.OptionRouter:     gatewayIP,
			dhcp4.OptionDomainNameServer: dnsBytes,
		}
		service.leases[macAddress] = newLease
		log.Printf("<dhcp> allocate new lease for MAC '%s', address %s/%s, gateway %s, dns %s, expire '%s'",
			macAddress, newLease.IP, newLease.Netmask, newLease.Gateway, strings.Join(newLease.DNS, "/"),
			newLease.Expire.Format(TimeFormatLayout))
		op.RespChan <- operateResult{IP: clientIP, Options: newLease.Options}
		return

	case opUpdate:
		var now = time.Now()
		var macAddress = op.MAC
		var requestIP = op.IP
		if lease, exists := service.leases[macAddress]; exists{
			if requestIP != lease.IP{
				err = fmt.Errorf("request IP '%s' diffrent from allocated '%s' for MAC '%s'",
					requestIP, lease.IP, macAddress)
				op.RespChan <- operateResult{Error:err}
				return
			}
			if now.After(lease.Expire){
				err = fmt.Errorf("lease of MAC '%s' expired ('%s')", macAddress, lease.Expire.Format(TimeFormatLayout))
				delete(service.leases, macAddress)
				op.RespChan <- operateResult{Error:err}
				return
			}
			//update
			lease.Expire = now.Add(leaseTimeout)
			service.leases[macAddress] = lease
			log.Printf("<dhcp> lease of MAC '%s' updated for request", macAddress)
			op.RespChan <- operateResult{Options: lease.Options}
			return
		}else{
			err = fmt.Errorf("no lease for MAC '%s'", macAddress)
			op.RespChan <- operateResult{Error:err}
			return
		}
	case opDeallocate:
		var macAddress = op.MAC
		if _, exists := service.leases[macAddress]; exists{
			log.Printf("<dhcp> lease of MAC '%s' released", macAddress)
			delete(service.leases, macAddress)
		}else{
			log.Printf("<dhcp> warning: no lease for MAC '%s' could release", macAddress)
		}
		return
	case opChange:
		gatewayIP, err := stringToIPv4(op.Gateway)
		if err != nil{
			log.Printf("<dhcp> parse gateway fail when update servers: %s", err.Error())
			return
		}
		var dnsBytes []byte
		for _, dns := range op.DNS{
			ip, err := stringToIPv4(dns)
			if err != nil{
				log.Printf("<dhcp> parse DNS fail when update servers: %s", err.Error())
				return
			}
			dnsBytes = append(dnsBytes, ip...)
		}
		for mac, lease := range service.leases{
			lease.Options[dhcp4.OptionRouter] = gatewayIP
			lease.Options[dhcp4.OptionDomainNameServer] = dnsBytes
			service.leases[mac] = lease
			log.Printf("<dhcp> servers of lease for '%s' changed", mac)
		}
	default:
		log.Printf("<dhcp> ignore invalid op type %d", op.Type)
	}
}

func (handler *DHCPHandler) ServeDHCP(req dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) (packet dhcp4.Packet){

	var err error
	switch msgType {
	case dhcp4.Discover:
		var macAddress = req.CHAddr().String()
		var respChan = make(chan operateResult, 1)
		handler.operates <- dhcpOperate{Type:opAllocate, MAC:macAddress, RespChan:respChan}
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("<dhcp> allocate lease for MAC '%s' fail: %s", macAddress, err.Error())
			return
		}
		return dhcp4.ReplyPacket(req, dhcp4.Offer, handler.serverIP, result.IP, leaseTimeout,
			result.Options.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))

	case dhcp4.Request:
		if requestServer, exists := options[dhcp4.OptionServerIdentifier]; exists{
			//check request server
			var serverIP = net.IP(requestServer)
			if !serverIP.Equal(handler.serverIP){
				//log.Printf("<dhcp> ignore request for different server '%s'", serverIP.String())
				return
			}
		}
		var requestIP = net.IP(options[dhcp4.OptionRequestedIPAddress])
		if requestIP == nil {
			requestIP = net.IP(req.CIAddr())
		}
		var macAddress = req.CHAddr().String()
		var respChan = make(chan operateResult, 1)
		handler.operates <- dhcpOperate{Type:opUpdate, MAC:macAddress, IP: requestIP.String(), RespChan:respChan}
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("<dhcp> update lease for MAC '%s' fail: %s", macAddress, err.Error())
			return dhcp4.ReplyPacket(req, dhcp4.NAK, handler.serverIP, nil, 0, nil)
		}
		return dhcp4.ReplyPacket(req, dhcp4.ACK, handler.serverIP, requestIP, leaseTimeout,
			result.Options.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
	case dhcp4.Release, dhcp4.Decline:
		var macAddress = req.CHAddr().String()
		handler.operates <- dhcpOperate{Type:opDeallocate, MAC:macAddress}
		return
	default:
		//log.Printf("<dhcp> ignore message type %d from %s", msgType, req.CHAddr().String())
	}
	return
}


func stringToIPv4(value string) (ip net.IP, err error){
	ip = net.ParseIP(value)
	if ip == nil{
		err = fmt.Errorf("invalid address '%s'", value)
		return
	}
	ip = ip.To4()
	if ip == nil{
		err = fmt.Errorf("invalid IPv4 address '%s'", value)
		return
	}
	return ip, nil
}


