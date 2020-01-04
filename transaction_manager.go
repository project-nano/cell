package main

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/cell/service"
	"github.com/project-nano/cell/task"
	"net/http"
	"crypto/tls"
	"math/rand"
	"time"
)

type TransactionManager struct {
	*framework.TransactionEngine
}

func CreateTransactionManager(sender framework.MessageSender, iManager *service.InstanceManager,
	sManager *service.StorageManager, nManager *service.NetworkManager) (*TransactionManager, error) {
	engine, err := framework.CreateTransactionEngine()
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	generator := rand.New(rand.NewSource(time.Now().UnixNano()))

	var manager = TransactionManager{engine}
	if err = manager.RegisterExecutor(framework.GetComputePoolCellRequest,
		&task.GetCellInfoExecutor{sender, iManager, sManager, nManager}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.CreateGuestRequest,
		&task.CreateInstanceExecutor{sender, iManager, sManager, nManager, generator}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteGuestRequest,
		&task.DeleteInstanceExecutor{sender, iManager, sManager, nManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetGuestRequest,
		&task.GetInstanceConfigExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetInstanceStatusRequest,
		&task.GetInstanceStatusExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.StartInstanceRequest,
		&task.StartInstanceExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.StopInstanceRequest,
		&task.StopInstanceExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.AttachInstanceRequest,
		&task.AttachInstanceExecutor{sender, iManager, sManager, nManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DetachInstanceRequest,
		&task.DetachInstanceExecutor{sender, iManager, sManager, nManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyGuestNameRequest,
		&task.ModifyGuestNameExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyCoreRequest,
		&task.ModifyGuestCoreExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyMemoryRequest,
		&task.ModifyGuestMemoryExecutor{sender, iManager}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.ModifyPriorityRequest,
		&task.ModifyCPUPriorityExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyDiskThresholdRequest,
		&task.ModifyDiskThresholdExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyNetworkThresholdRequest,
		&task.ModifyNetworkThresholdExecutor{sender, iManager}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.ModifyAuthRequest,
		&task.ModifyGuestPasswordExecutor{sender, iManager, generator}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetAuthRequest,
		&task.GetGuestPasswordExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ResetSystemRequest,
		&task.ResetGuestSystemExecutor{sender, iManager, sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.InsertMediaRequest,
		&task.InsertMediaCoreExecutor{sender, iManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.EjectMediaRequest,
		&task.EjectMediaCoreExecutor{sender, iManager}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.ComputePoolReadyEvent,
		&task.HandleComputePoolReadyExecutor{sender, iManager, sManager, nManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ComputeCellRemovedEvent,
		&task.HandleComputeCellRemovedExecutor{sender, iManager, sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateDiskImageRequest,
		&task.CreateDiskImageExecutor{sender, iManager, sManager, client}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ResizeDiskRequest,
		&task.ResizeGuestVolumeExecutor{sender, iManager, sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ShrinkDiskRequest,
		&task.ShrinkGuestVolumeExecutor{sender, iManager, sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QuerySnapshotRequest,
		&task.QuerySnapshotExecutor{sender, sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetSnapshotRequest,
		&task.GetSnapshotExecutor{sender,  sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.AddressPoolChangedEvent,
		&task.HandleAddressPoolChangedExecutor{  nManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateSnapshotRequest,
		&task.CreateSnapshotExecutor{sender, iManager, sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteSnapshotRequest,
		&task.DeleteSnapshotExecutor{sender, iManager, sManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.RestoreSnapshotRequest,
		&task.RestoreSnapshotExecutor{sender, iManager, sManager}); err != nil{
		return nil, err
	}
	return &manager, nil
}
