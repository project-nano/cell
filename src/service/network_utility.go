package service

import (
	"net"
	"fmt"
	"github.com/libvirt/libvirt-go"
	"encoding/xml"
)

const (
	DefaultBridgeName = "br0"
	DefaultNetworkName = "default"
)

type NetworkUtility struct {
	virConnect *libvirt.Connect
}

type virNetworkRange struct {
	Start string `xml:"start,attr"`
	End   string `xml:"end,attr"`
}

type virNetworkNAT struct {
	Port    *virNetworkRange `xml:"port,omitempty"`
	Address *virNetworkRange `xml:"address,omitempty"`
}

type virNetworkForward struct {
	NAT  *virNetworkNAT `xml:"nat,omitempty"`
	Mode string         `xml:"mode,attr"`
}

type virNetworkBridge struct {
	Name  string `xml:"name,attr"`
	STP   string `xml:"stp,attr,omitempty"`
	Delay string `xml:"delay,attr,omitempty"`
}

type virNetworkMAC struct {
	Address string `xml:"address,attr,omitempty"`
}

type virNetworkDHCP struct {
	Range *virNetworkRange `xml:"range,omitempty"`
}

type virNetworkIP struct {
	Address string `xml:"address,attr"`
	Netmask string `xml:"netmask,attr"`
	DHCP *virNetworkDHCP `xml:"dhcp,omitempty"`
}

type virNetworkDefine struct {
	XMLName xml.Name           `xml:"network"`
	Name    string             `xml:"name"`
	UUID    string             `xml:"uuid,omitempty"`
	Forward *virNetworkForward `xml:"forward,omitempty"`
	Bridge  *virNetworkBridge  `xml:"bridge,omitempty"`
	MAC     *virNetworkMAC     `xml:"mac,omitempty"`
	IP      *virNetworkIP      `xml:"ip,omitempty"`
}

func (util *NetworkUtility) DisableDHCPonDefaultNetwork() (changed bool, err error) {
	changed = false
	virNetwork, err := util.virConnect.LookupNetworkByName(DefaultNetworkName)
	if err != nil{
		return changed, err
	}
	desc, err := virNetwork.GetXMLDesc(0)
	if err != nil{
		return changed, err
	}
	var define virNetworkDefine
	if err = xml.Unmarshal([]byte(desc), &define); err != nil{
		return changed, err
	}
	if nil == define.IP{
		return changed, nil
	}
	if nil == define.IP.DHCP{
		return changed, nil
	}

	activated, err := virNetwork.IsActive()
	if err != nil{
		return changed, err
	}
	if activated {
		if err = virNetwork.Destroy(); err != nil{
			return
		}
	}
	//disable dhcp
	define.IP.DHCP = nil
	newConfig, err := xml.MarshalIndent(define, "", "")
	if err != nil{
		return
	}
	virNetwork, err = util.virConnect.NetworkDefineXML(string(newConfig))
	if err != nil{
		return
	}
	changed = true
	if activated{
		if err = virNetwork.Create();err != nil{
			return
		}
	}
	return  changed,nil
}

var ipOfDefaultBridge = ""

func GetCurrentIPOfDefaultBridge() (ip string, err error){
	if "" != ipOfDefaultBridge{
		return ipOfDefaultBridge, nil
	}
	dev, err := net.InterfaceByName(DefaultBridgeName)
	if err != nil{
		return  "", err
	}
	addrs, err := dev.Addrs()
	if err != nil{
		return "", err
	}
	for _, addr := range addrs{
		var CIDRString = addr.String()
		ip, _, err := net.ParseCIDR(CIDRString)
		if err != nil {
			return "",err
		}
		var ipv4 = ip.To4()
		if ipv4 != nil {
			//v4 ip only
			ipOfDefaultBridge = ip.String()
			return ipOfDefaultBridge, nil
		}
	}
	return "", fmt.Errorf("no ipv4 address available in %s", DefaultBridgeName)
}
