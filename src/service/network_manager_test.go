package service

import (
	"github.com/libvirt/libvirt-go"
	"os"
	"testing"
)

const (
	testPathForNetworkManager = "../../test/network"
)

func clearTestEnvironmentForNetworkManager() (err error) {
	if _, err = os.Stat(testPathForNetworkManager); !os.IsNotExist(err) {
		//exists
		err = os.RemoveAll(testPathForNetworkManager)
		return
	}
	return nil
}

func getNetworkManagerForTest() (manager *NetworkManager, err error) {
	const (
		libvirtURL     = "qemu:///system"
		maxMonitorPort = 50
	)
	if err = clearTestEnvironmentForNetworkManager(); err != nil {
		return
	}
	if err = os.MkdirAll(testPathForNetworkManager, 0740); err != nil {
		return
	}
	// initial libvirt connection
	var virConnect *libvirt.Connect
	if virConnect, err = libvirt.NewConnect(libvirtURL); err != nil {
		return
	}
	if manager, err = CreateNetworkManager(testPathForNetworkManager, virConnect, maxMonitorPort); err != nil {
		return
	}
	err = manager.Start()
	return
}

func TestNetworkManager_AllocateResource(t *testing.T) {
	const (
		instanceID      = "sample"
		macAddress      = "00:11:22:33:44:55"
		internalAddress = ""
		externalAddress = ""
	)
	var manager *NetworkManager
	var err error
	if manager, err = getNetworkManagerForTest(); err != nil {
		t.Fatalf("get network manager fail: %s", err.Error())
		return
	}
	defer func() {
		if err = manager.Stop(); err != nil {
			t.Error(err)
		}
	}()
	//allocate
	respChan := make(chan NetworkResult, 1)
	manager.AllocateInstanceResource(instanceID, macAddress, internalAddress, externalAddress, respChan)
	var result = <-respChan
	if result.Error != nil {
		t.Fatalf("allocate resource fail: %s", result.Error.Error())
	}
	var monitorPort = result.MonitorPort
	t.Logf("monitor port %d allocated", monitorPort)
	t.Log("test network manager allocate resource success")
}
