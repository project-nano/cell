package service

import (
	"fmt"
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
	var verifySnapshotTree = func(checkpoint string, required [][]expectSnapshot) {
		// query and print snapshot tree
		queryChan := make(chan StorageResult, 1)
		manager.QuerySnapshot(volumeName, queryChan)
		resp := <-queryChan
		if resp.Error != nil {
			t.Fatalf("query snapshot failed: %s", resp.Error.Error())
		}
		dumpSnapshots(resp.SnapshotList, t.Logf)
		if err = checkSnapshotTree(resp.SnapshotList, required); err != nil {
			t.Fatalf("checkpoint %s failed: %s", checkpoint, err.Error())
		}
		t.Logf("checkpoint %s passed", checkpoint)
	}
	batchDelete := func(targets []string) {
		respChan := make(chan error, 1)
		for _, snapshotName := range targets {
			manager.DeleteSnapshot(volumeName, snapshotName, respChan)
			err = <-respChan
			if nil != err {
				t.Fatalf("delete snapshot %s failed: %s", snapshotName, err.Error())
			} else {
				t.Logf("delete snapshot %s success", snapshotName)
			}
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
	}
	// checkpoint 1
	{
		// s111 -> s11 -> s1 -> s0(root)
		// s112 -> s11 -> s1 -> s0(root)
		// s211 -> s21 -> s2 -> s0(root)
		// s3(current) -> s0(root)
		var checkpoint1 = [][]expectSnapshot{
			{
				{name: s111, isCurrent: false, isRoot: false},
				{name: s11, isCurrent: false, isRoot: false},
				{name: s1, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
			{
				{name: s112, isCurrent: false, isRoot: false},
				{name: s11, isCurrent: false, isRoot: false},
				{name: s1, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
			{
				{name: s211, isCurrent: false, isRoot: false},
				{name: s21, isCurrent: false, isRoot: false},
				{name: s2, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
			{
				{name: s3, isCurrent: true, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
		}
		verifySnapshotTree("checkpoint 1", checkpoint1)
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
		batches := []string{
			s112,
			s2,
			s3,
		}
		batchDelete(batches)
	}
	{
		// s111 -> s11 -> s1 -> s0(root, current)
		// s11 -> s1 -> s0(root, current)
		// s211 -> s21 -> s0(root, current)
		var checkpoint2 = [][]expectSnapshot{
			{
				{name: s111, isCurrent: false, isRoot: false},
				{name: s11, isCurrent: false, isRoot: false},
				{name: s1, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: true, isRoot: true},
			},
			{
				{name: s11, isCurrent: false, isRoot: false},
				{name: s1, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: true, isRoot: true},
			},
			{
				{name: s211, isCurrent: false, isRoot: false},
				{name: s21, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: true, isRoot: true},
			},
		}
		verifySnapshotTree("checkpoint 2", checkpoint2)
	}
	{
		// create s3 to current
		var targetName = s3
		respChan := make(chan error, 1)
		manager.CreateSnapshot(volumeName, targetName, targetName, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("create snapshot %s failed: %s", targetName, err.Error())
		}
		t.Logf("snapshot %s created", targetName)
		// revert to s11
		targetName = s11
		manager.RestoreSnapshot(volumeName, targetName, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("restore snapshot %s failed: %s", targetName, err.Error())
		}
		t.Logf("snapshot %s restored", targetName)
		// create s112 to current
		targetName = s112
		manager.CreateSnapshot(volumeName, targetName, targetName, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("create snapshot %s failed: %s", targetName, err.Error())
		}
		t.Logf("snapshot %s created", targetName)
	}
	{
		//checkpoint 3
		// s111 -> s11 -> s1 -> s0(root)
		// s112(current) -> s11 -> s1 -> s0(root)
		// s211 -> s21 -> s0(root)
		// s3-> s0(root)
		var checkpoint3 = [][]expectSnapshot{
			{
				{name: s111, isCurrent: false, isRoot: false},
				{name: s11, isCurrent: false, isRoot: false},
				{name: s1, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
			{
				{name: s112, isCurrent: true, isRoot: false},
				{name: s11, isCurrent: false, isRoot: false},
				{name: s1, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
			{
				{name: s211, isCurrent: false, isRoot: false},
				{name: s21, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
			{
				{name: s3, isCurrent: false, isRoot: false},
				{name: s0, isCurrent: false, isRoot: true},
			},
		}
		verifySnapshotTree("checkpoint 3", checkpoint3)
	}
	{
		batches := []string{
			s111,
			s112,
			s11,
			s1,
			s21,
			s211,
			s3,
			s0,
		}
		batchDelete(batches)
	}
	{
		// create snapshot s3, s2
		var targetName = s3
		respChan := make(chan error, 1)
		manager.CreateSnapshot(volumeName, targetName, targetName, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("create snapshot %s failed: %s", targetName, err.Error())
		}
		t.Logf("snapshot %s created", targetName)
		targetName = s2
		manager.CreateSnapshot(volumeName, targetName, targetName, respChan)
		if err = <-respChan; err != nil {
			t.Fatalf("create snapshot %s failed: %s", targetName, err.Error())
		}
		t.Logf("snapshot %s created", targetName)
	}
	{
		//checkpoint 4
		// s2(current) -> s3(root)
		var checkpoint4 = [][]expectSnapshot{
			{
				{name: s2, isCurrent: true, isRoot: false},
				{name: s3, isCurrent: false, isRoot: true},
			},
		}
		verifySnapshotTree("checkpoint 4", checkpoint4)

	}
	t.Log("test case: Snapshots passed")
}

type logPrint func(format string, v ...interface{})

func dumpSnapshots(snapshots []SnapshotConfig, log logPrint) {
	for _, node := range snapshots {
		// dump snapshot name, backing, current and root flag
		log("snapshot %s, backing %s, current %v, root %v", node.Name, node.Backing, node.IsCurrent, node.IsRoot)
	}
}

type expectSnapshot struct {
	name      string
	isCurrent bool
	isRoot    bool
}

func checkSnapshotTree(snapshots []SnapshotConfig, required [][]expectSnapshot) (err error) {
	//build snapshot map
	var snapshotMap = map[string]SnapshotConfig{}
	for _, snapshot := range snapshots {
		snapshotMap[snapshot.Name] = snapshot
	}
	for seqIndex, sequence := range required {
		seqLength := len(sequence)
		for index := 0; index < seqLength; index++ {
			depth := seqLength - index - 1
			expected := sequence[index]
			snapshot, exists := snapshotMap[expected.name]
			if !exists {
				err = fmt.Errorf("cannot find snapshot %s for sequence %d, depth %d",
					expected.name, seqIndex, depth)
				return
			}
			if snapshot.IsCurrent != expected.isCurrent {
				err = fmt.Errorf("snapshot %s current flag mismatch for sequence %d, depth %d. expected %v, actual %v",
					expected.name, seqIndex, depth, expected.isCurrent, snapshot.IsCurrent)
				return
			}
			if snapshot.IsRoot != expected.isRoot {
				err = fmt.Errorf("snapshot %s root flag mismatch for sequence %d, depth %d. expected %v, actual %v",
					expected.name, seqIndex, depth, expected.isRoot, snapshot.IsRoot)
				return
			}
			// check backing
			if depth > 0 {
				backing := sequence[index+1]
				if snapshot.Backing != backing.name {
					err = fmt.Errorf("snapshot %s backing mismatch for sequence %d, depth %d. expected %s, actual %s",
						expected.name, seqIndex, depth, backing.name, snapshot.Backing)
					return
				}
			}
		}
	}
	return nil
}
