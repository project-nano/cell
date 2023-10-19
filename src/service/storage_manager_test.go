package service

import (
	"github.com/libvirt/libvirt-go"
	"os"
	"testing"
)

const (
	testPathForStorageManager = "../../test/storage"
)

func clearStorageManagerTestEnvironment() {
	if _, err := os.Stat(testPathForStorageManager); !os.IsNotExist(err) {
		_ = os.RemoveAll(testPathForStorageManager)
	}
}

func getStorageManagerForTest() (manager *StorageManager, err error) {
	const (
		libvirtURL = "qemu:///system"
	)
	clearStorageManagerTestEnvironment()
	if err = os.MkdirAll(testPathForStorageManager, 0755); err != nil {
		return
	}
	var virConnect *libvirt.Connect
	if virConnect, err = libvirt.NewConnect(libvirtURL); err != nil {
		return
	}
	if manager, err = CreateStorageManager(testPathForStorageManager, virConnect); err != nil {
		return
	}
	err = manager.Start()
	return
}

func TestStorageManager_CreateVolumes(t *testing.T) {
	manager, err := getStorageManagerForTest()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = manager.Stop()
	}()
	const (
		volumeName = "test_volume"
		gib        = 1 << 30
	)
	var system uint64 = 5 * gib
	var data = []uint64{
		2 * gib,
		1 * gib,
	}
	{
		respChan := make(chan StorageResult, 1)
		manager.CreateVolumes(volumeName, system, data, BootTypeNone, respChan)
		var resp = <-respChan
		if resp.Error != nil {
			t.Fatal(resp.Error)
		}
		t.Logf("vol %s created", volumeName)
	}
	{
		respChan := make(chan error, 1)
		manager.DeleteVolumes(volumeName, respChan)
		err = <-respChan
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("vol %s deleted", volumeName)
	}
	t.Log("test case: CreateVolumes passed")
}

func TestStorageManager_Snapshots(t *testing.T) {
	manager, err := getStorageManagerForTest()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = manager.Stop()
	}()
	const (
		volumeName = "test_volume"
		gib        = 1 << 30
	)
	var system uint64 = 5 * gib
	var data = []uint64{
		2 * gib,
		1 * gib,
	}
	{
		respChan := make(chan StorageResult, 1)
		manager.CreateVolumes(volumeName, system, data, BootTypeNone, respChan)
		var resp = <-respChan
		if resp.Error != nil {
			t.Fatal(resp.Error)
		}
		t.Logf("vol %s created", volumeName)
	}
	defer func() {
		respChan := make(chan error, 1)
		manager.DeleteVolumes(volumeName, respChan)
		err = <-respChan
		if err != nil {
			t.Logf("warning: delete vol %s failed: %s", volumeName, err.Error())
		}
		t.Logf("vol %s deleted", volumeName)
	}()
	// initial snapshot tree
	const (
		s0   = "root"
		s1   = "s1"
		s11  = "s1.1"
		s111 = "s1.1-alpha"
		s112 = "s1.1-beta"
		s2   = "s2"
		s21  = "s2.1"
		s211 = "omega"
		s3   = "omicron"
	)
	var dumpSnapshots = func() {
		// query and print snapshot tree
		queryChan := make(chan StorageResult, 1)
		manager.QuerySnapshot(volumeName, queryChan)
		resp := <-queryChan
		if resp.Error != nil {
			t.Fatalf("query snapshot failed: %s", resp.Error.Error())
		}
		for _, node := range resp.SnapshotList {
			// dump snapshot name, backing, current and root flag
			t.Logf("snapshot %s, backing %s, current %v, root %v", node.Name, node.Backing, node.IsCurrent, node.IsRoot)
		}
	}
	{
		//initial s0->s1->s11->s111
		serial := []string{s0, s1, s11, s111}
		respChan := make(chan error, 1)
		for _, name := range serial {
			manager.CreateSnapshot(volumeName, name, name, respChan)
			err = <-respChan
			if err != nil {
				t.Fatalf("create snapshot %s failed: %s", name, err.Error())
			}
			t.Logf("snapshot %s created", name)
		}
		// restore to s11
		manager.RestoreSnapshot(volumeName, s11, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("restore snapshot %s failed: %s", s11, err.Error())
		}
		t.Logf("snapshot %s restored", s11)
		// create s112
		manager.CreateSnapshot(volumeName, s112, s112, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("create snapshot %s failed: %s", s112, err.Error())
		}
		t.Logf("snapshot %s created", s112)
		// restore s0
		manager.RestoreSnapshot(volumeName, s0, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("restore snapshot %s failed: %s", s0, err.Error())
		}
		t.Logf("snapshot %s restored", s0)
		// serial : s2 -> s21-> s211
		serial = []string{s2, s21, s211}
		for _, name := range serial {
			manager.CreateSnapshot(volumeName, name, name, respChan)
			err = <-respChan
			if err != nil {
				t.Fatalf("create snapshot %s failed: %s", name, err.Error())
			}
			t.Logf("snapshot %s created", name)
		}
		// restore to s0
		manager.RestoreSnapshot(volumeName, s0, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("restore snapshot %s failed: %s", s0, err.Error())
		}
		t.Logf("snapshot %s restored", s0)
		// serial : s3
		serial = []string{s3}
		for _, name := range serial {
			manager.CreateSnapshot(volumeName, name, name, respChan)
			err = <-respChan
			if err != nil {
				t.Fatalf("create snapshot %s failed: %s", name, err.Error())
			}
			t.Logf("snapshot %s created", name)
		}
		dumpSnapshots()
	}
	{
		// delete s0 and must fail
		respChan := make(chan error, 1)
		manager.DeleteSnapshot(volumeName, s0, respChan)
		if err = <-respChan; err == nil {
			t.Fatalf("delete snapshot %s should fail", s0)
		}
		t.Logf("delete snapshot %s failed as expected: %s", s0, err.Error())
	}
	{
		// delete s2 and must fail
		respChan := make(chan error, 1)
		manager.DeleteSnapshot(volumeName, s2, respChan)
		if err = <-respChan; err == nil {
			t.Fatalf("delete snapshot %s should fail", s2)
		}
		t.Logf("delete snapshot %s failed as expected: %s", s2, err.Error())
	}
	{
		// delete s211 and must success
		respChan := make(chan error, 1)
		manager.DeleteSnapshot(volumeName, s211, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("delete snapshot %s failed: %s", s211, err.Error())
		}
		t.Logf("delete snapshot %s success", s211)
	}
	{
		// delete s21 and must success
		respChan := make(chan error, 1)
		manager.DeleteSnapshot(volumeName, s21, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("delete snapshot %s failed: %s", s21, err.Error())
		}
		t.Logf("delete snapshot %s success", s21)
	}
	{
		// delete s3 and must fail
		respChan := make(chan error, 1)
		manager.DeleteSnapshot(volumeName, s3, respChan)
		if err = <-respChan; err == nil {
			t.Fatalf("delete snapshot %s should fail", s3)
		}
		t.Logf("delete snapshot %s failed as expected: %s", s3, err.Error())
	}
	dumpSnapshots()
	t.Log("test case: Snapshots passed")
}
