package service

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/project-nano/framework"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type SchedulerResult struct {
	Error error
	ID    framework.SessionID
	Size  uint
}

type SchedulerUpdate struct {
	ID       framework.SessionID
	Progress uint
}

type scheduleTask struct {
	Type       scheduleTaskType
	ID         framework.SessionID
	Path       string
	Image      string
	Host       string
	Port       uint
	ImageSize  uint64
	TargetSize uint64
	Group      string
	Volume     string
	Snapshot   string
	Targets    []snapshotTarget
	ErrorChan  chan error
}

type scheduleTaskType int

const (
	scheduleWriteDiskImage = iota
	scheduleReadDiskImage
	scheduleResizeDisk
	scheduleShrinkDisk
	scheduleCreateSnapshot
	scheduleRestoreSnapshot
	scheduleDeleteSnapshot
)

type schedulerEventType int

const (
	schedulerEventWriteDiskCompleted = schedulerEventType(iota)
	schedulerEventReadDiskCompleted
	schedulerEventShrinkDiskCompleted
	schedulerEventResizeDiskCompleted
	schedulerEventCreateSnapshotCompleted
	schedulerEventDeleteSnapshotCompleted
	schedulerEventRestoreSnapshotCompleted
)

type schedulerEvent struct {
	Type      schedulerEventType
	Group     string
	Volume    string
	Snapshot  string
	Error     error
	ErrorChan chan error
}

type snapshotTarget struct {
	Current string `json:"current"`
	Backing string `json:"backing,omitempty"`
	Backed  bool   `json:"backed,omitempty"`
}

type IOScheduler struct {
	name         string
	progressChan chan SchedulerUpdate
	resultChan   chan SchedulerResult
	taskChan     chan scheduleTask
	eventChan    chan schedulerEvent
	client       *http.Client
	runner       *framework.SimpleRunner
}

func CreateScheduler(poolName string, progressChan chan SchedulerUpdate, resultChan chan SchedulerResult, eventChan chan schedulerEvent) (scheduler *IOScheduler, err error) {
	const (
		DefaultQueueSize = 1 << 10
	)
	scheduler = &IOScheduler{}
	scheduler.name = poolName
	scheduler.progressChan = progressChan
	scheduler.resultChan = resultChan
	scheduler.eventChan = eventChan
	scheduler.taskChan = make(chan scheduleTask, DefaultQueueSize)
	scheduler.runner = framework.CreateSimpleRunner(scheduler.Routine)
	scheduler.client = &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	return scheduler, nil
}

func (scheduler *IOScheduler) Start() error {
	return scheduler.runner.Start()
}

func (scheduler *IOScheduler) Stop() error {
	return scheduler.runner.Stop()
}

func (scheduler *IOScheduler) Routine(c framework.RoutineController) {
	log.Printf("<scheduler-%s> started", scheduler.name)
	for !c.IsStopping() {
		select {
		case <-c.GetNotifyChannel():
			c.SetStopping()
		case task := <-scheduler.taskChan:
			scheduler.handleTask(task)
		}
	}
	c.NotifyExit()
	log.Printf("<scheduler-%s> stopped", scheduler.name)
}

func (scheduler *IOScheduler) AddWriteTask(id framework.SessionID, group, volume, path, image, host string, port uint) {
	task := scheduleTask{Type: scheduleWriteDiskImage, ID: id, Group: group, Volume: volume, Path: path, Image: image, Host: host, Port: port}
	scheduler.taskChan <- task
}

func (scheduler *IOScheduler) AddReadTask(id framework.SessionID, group, volume, path, image string, targetSize, imgSize uint64, host string, port uint) {
	task := scheduleTask{Type: scheduleReadDiskImage, ID: id, Group: group, Volume: volume, Path: path, Image: image, ImageSize: imgSize, TargetSize: targetSize, Host: host, Port: port}
	scheduler.taskChan <- task
}

func (scheduler *IOScheduler) AddResizeTask(id framework.SessionID, group, volume, path string, size uint64) {
	scheduler.taskChan <- scheduleTask{Type: scheduleResizeDisk, ID: id, Group: group, Volume: volume, Path: path, TargetSize: size}
}

func (scheduler *IOScheduler) AddShrinkTask(id framework.SessionID, group, volume, path string) {
	scheduler.taskChan <- scheduleTask{Type: scheduleShrinkDisk, ID: id, Group: group, Volume: volume, Path: path}
}

func (scheduler *IOScheduler) AddCreateSnapshotTask(group, snapshot string, targets []snapshotTarget, respChan chan error) {
	scheduler.taskChan <- scheduleTask{Type: scheduleCreateSnapshot, Group: group, Snapshot: snapshot, Targets: targets, ErrorChan: respChan}
}

func (scheduler *IOScheduler) AddRestoreSnapshotTask(group, snapshot string, targets []snapshotTarget, respChan chan error) {
	scheduler.taskChan <- scheduleTask{Type: scheduleRestoreSnapshot, Group: group, Snapshot: snapshot, Targets: targets, ErrorChan: respChan}
}

func (scheduler *IOScheduler) AddDeleteSnapshotTask(group, snapshot string, targets []snapshotTarget, respChan chan error) {
	scheduler.taskChan <- scheduleTask{Type: scheduleDeleteSnapshot, Group: group, Snapshot: snapshot, Targets: targets, ErrorChan: respChan}
}

func (scheduler *IOScheduler) handleTask(task scheduleTask) {
	var err error
	switch task.Type {
	case scheduleReadDiskImage:
		err = scheduler.handleReadTask(task.ID, task.Group, task.Volume, task.Path, task.Image, task.ImageSize, task.TargetSize, task.Host, task.Port)
	case scheduleWriteDiskImage:
		err = scheduler.handleWriteTask(task.ID, task.Group, task.Volume, task.Path, task.Image, task.Host, task.Port)
	case scheduleResizeDisk:
		err = scheduler.handleResizeTask(task.ID, task.Group, task.Volume, task.Path, task.TargetSize)
	case scheduleShrinkDisk:
		err = scheduler.handleShrinkTask(task.ID, task.Group, task.Volume, task.Path)
	case scheduleCreateSnapshot:
		err = scheduler.handleCreateSnapshotTask(task.Group, task.Snapshot, task.Targets, task.ErrorChan)
	case scheduleDeleteSnapshot:
		err = scheduler.handleDeleteSnapshotTask(task.Group, task.Snapshot, task.Targets, task.ErrorChan)
	case scheduleRestoreSnapshot:
		err = scheduler.handleRestoreSnapshotTask(task.Group, task.Snapshot, task.Targets, task.ErrorChan)
	default:
		log.Printf("<scheduler-%s> ignore unsupported task type %d", scheduler.name, task.Type)
	}
	if err != nil {
		log.Printf("<scheduler-%s> handle task fail: %s, type %d, id %08X", scheduler.name, err.Error(),
			task.Type, task.ID)
	}
}

func (scheduler *IOScheduler) handleWriteTask(id framework.SessionID, group, volume, path, image, host string, port uint) (err error) {
	const (
		Protocol       = "https"
		Resource       = "disk_images"
		FieldName      = "image"
		CheckSumField  = "checksum"
		ChunkSize      = 1 << 10
		NotifyInterval = 1 * time.Second
	)
	defer func(e error) {
		var event = schedulerEvent{Type: schedulerEventWriteDiskCompleted, Group: group, Volume: volume}
		event.Error = e
		scheduler.eventChan <- event
	}(err)
	var url = fmt.Sprintf("%s://%s:%d%s", Protocol, host, port,
		scheduler.apiPath(fmt.Sprintf("/%s/%s/file/", Resource, image)))
	stat, err := os.Stat(path)
	if err != nil {
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}
	var totalSize = int(stat.Size())
	checkSum, err := scheduler.generateCheckSum(path)
	if err != nil {
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}

	var processed = 0
	var pipeFinished = make(chan bool)
	bodyReader, bodyWriter := io.Pipe()

	var multiWriter = multipart.NewWriter(bodyWriter)
	var contentType = multiWriter.FormDataContentType()

	go func() {
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("<scheduler-%s> close file fail: %s", scheduler.name, err.Error())
			}
			if err := bodyWriter.Close(); err != nil {
				log.Printf("<scheduler-%s> close bodywriter fail: %s", scheduler.name, err.Error())
			}
			pipeFinished <- true
		}()
		multiWriter.WriteField(CheckSumField, checkSum)
		filePartWriter, err := multiWriter.CreateFormFile(FieldName, image)
		if err != nil {
			log.Printf("<scheduler-%s> create writer field fail: %s", scheduler.name, err.Error())
			return
		}
		defer multiWriter.Close()

		var n int
		var buffer = make([]byte, ChunkSize)

		for {
			n, err = file.Read(buffer)
			if n > 0 {
				filePartWriter.Write(buffer[:n])
				processed += n
			}
			if err == io.EOF {
				//log.Printf("<scheduler-%s> finished reading file '%s'", scheduler.name, path)
				return
			} else if err != nil {
				log.Printf("<scheduler-%s> reading file fail: %s", scheduler.name, err.Error())
				return
			}
		}
	}()

	var notifyChan = make(chan bool)
	var exitChan = make(chan bool)
	go func() {
		ticker := time.NewTicker(NotifyInterval)
		for {
			select {
			case <-notifyChan:
				exitChan <- true
				return
			case <-ticker.C:
				var progress = processed * 100 / totalSize
				if progress > 100 {
					progress = 100
				}
				scheduler.progressChan <- SchedulerUpdate{Progress: uint(progress), ID: id}
			}
		}
	}()
	var finishRoutine = func() {
		notifyChan <- true
		<-exitChan
		<-pipeFinished
	}
	req, err := http.NewRequest(http.MethodPut, url, bodyReader)
	if err != nil {
		finishRoutine()
		scheduler.resultChan <- SchedulerResult{ID: id, Error: err}
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := scheduler.client.Do(req)
	finishRoutine()
	if err != nil {
		scheduler.resultChan <- SchedulerResult{ID: id, Error: err}
		return err
	}
	defer resp.Body.Close()
	type serverResponse struct {
		ErrorCode int    `json:"error_code"`
		Message   string `json:"message"`
	}
	var jsonResp serverResponse
	var decoder = json.NewDecoder(resp.Body)
	if err = decoder.Decode(&jsonResp); err != nil {
		scheduler.resultChan <- SchedulerResult{ID: id, Error: err}
		return err
	}
	if 0 != jsonResp.ErrorCode {
		err = errors.New(jsonResp.Message)
		scheduler.resultChan <- SchedulerResult{ID: id, Error: err}
		return err
	}
	scheduler.resultChan <- SchedulerResult{ID: id, Size: uint(totalSize)}
	return nil
}

func (scheduler *IOScheduler) handleReadTask(id framework.SessionID, group, volume, path, image string, imageSize, targetSize uint64, host string, port uint) (err error) {
	const (
		Protocol       = "https"
		Resource       = "disk_images"
		ChunkSize      = 1 << 10
		NotifyInterval = 1 * time.Second
		VolumePerm     = 0666
	)
	defer func(e error) {
		var event = schedulerEvent{Type: schedulerEventReadDiskCompleted, Group: group, Volume: volume}
		event.Error = e
		scheduler.eventChan <- event
	}(err)
	var url = fmt.Sprintf("%s://%s:%d%s", Protocol, host, port,
		scheduler.apiPath(fmt.Sprintf("/%s/%s/file/", Resource, image)))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, VolumePerm)
	if err != nil {
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}
	defer file.Close()

	resp, err := scheduler.client.Get(url)
	if err != nil {
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}
	defer resp.Body.Close()

	var checkSum = resp.Header.Get("Signature")

	var processed = 0
	var notifyChan = make(chan bool)
	var exitChan = make(chan bool)
	go func() {
		ticker := time.NewTicker(NotifyInterval)
		for {
			select {
			case <-notifyChan:
				exitChan <- true
				return
			case <-ticker.C:
				var progress = processed * 100 / int(imageSize)
				if progress > 100 {
					progress = 100
				}
				scheduler.progressChan <- SchedulerUpdate{Progress: uint(progress), ID: id}
			}
		}
	}()

	{
		var n int
		var buffer = make([]byte, ChunkSize)
		for {
			n, err = resp.Body.Read(buffer)
			if n > 0 {
				if n, wErr := file.Write(buffer[:n]); wErr != nil {
					log.Printf("<scheduler-%s> write to file fail: %s", scheduler.name, wErr.Error())
					scheduler.resultChan <- SchedulerResult{Error: wErr, ID: id}
					break
				} else {
					processed += n
				}
			}
			if err == io.EOF {
				//finish
				{
					if "" != checkSum {
						log.Printf("<scheduler-%s> all image data received, checking integrity...", scheduler.name)
						computed, err := scheduler.generateCheckSum(path)
						if err != nil {
							log.Printf("<scheduler-%s> generate check sum fail: %s", scheduler.name, err.Error())
							os.Remove(path)
							scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
							break
						}
						if checkSum != computed {
							err = errors.New("image corrupted")
							log.Printf("<scheduler-%s> check integrity fail: %s", scheduler.name, err.Error())
							scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
							break
						}
					}
					//resize image
					cmd := exec.Command("qemu-img", "resize", path, fmt.Sprintf("%d", targetSize))
					if err = cmd.Run(); err != nil {
						log.Printf("<scheduler-%s> resize image fail: %s", scheduler.name, err.Error())
						os.Remove(path)
						err = fmt.Errorf("resize image to %s fail: %s", bytesToString(targetSize), err.Error())
						scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
						break
					}
					log.Printf("<scheduler-%s> image %s resized to %s", scheduler.name, path, bytesToString(targetSize))
				}
				scheduler.resultChan <- SchedulerResult{ID: id, Size: uint(processed)}
				break
			} else if err != nil {
				log.Printf("<scheduler-%s> reading stream fail: %s", scheduler.name, err.Error())
				scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
				break
			}
		}

	}
	notifyChan <- true
	<-exitChan

	return nil
}

func (scheduler *IOScheduler) handleResizeTask(id framework.SessionID, group, volume, path string, size uint64) (err error) {
	defer func(e error) {
		var event = schedulerEvent{Type: schedulerEventResizeDiskCompleted, Group: group, Volume: volume}
		event.Error = e
		scheduler.eventChan <- event
	}(err)
	var begin = time.Now()
	var cmd = exec.Command("qemu-img", "resize", path, fmt.Sprintf("%d", size))
	log.Printf("<scheduler-%s> try resize volume %s ...", scheduler.name, path)
	errMessage, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.New(string(errMessage))
		log.Printf("<scheduler-%s> resize volume fail: %s", scheduler.name, err.Error())
		err = fmt.Errorf("resize volume to %s fail: %s", bytesToString(size), err.Error())
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}
	var elapsed = time.Now().Sub(begin).Seconds() * 1000
	scheduler.resultChan <- SchedulerResult{ID: id}
	log.Printf("<scheduler-%s> resize volume '%s' to %s in %.3f milliseconds",
		scheduler.name, path, bytesToString(size), elapsed)
	return nil
}

func (scheduler *IOScheduler) handleShrinkTask(id framework.SessionID, group, volume, path string) (err error) {
	defer func(e error) {
		var event = schedulerEvent{Type: schedulerEventShrinkDiskCompleted, Group: group, Volume: volume}
		event.Error = e
		scheduler.eventChan <- event
	}(err)
	var begin = time.Now()
	var shrankPath = fmt.Sprintf("%s_shrink", path)
	var cmd = exec.Command("qemu-img", "convert", "-f", "qcow2", "-O", "qcow2", path, shrankPath)
	log.Printf("<scheduler-%s> try shrink volume %s ...", scheduler.name, path)
	errMessage, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.New(string(errMessage))
		log.Printf("<scheduler-%s> shrink volume fail: %s", scheduler.name, err.Error())
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}
	log.Printf("<scheduler-%s> shrank volume %s created", scheduler.name, shrankPath)
	if err = os.Remove(path); err != nil {
		log.Printf("<scheduler-%s> remove old image fail: %s", scheduler.name, err.Error())
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}
	if err = os.Rename(shrankPath, path); err != nil {
		log.Printf("<scheduler-%s> rename new image fail: %s", scheduler.name, err.Error())
		scheduler.resultChan <- SchedulerResult{Error: err, ID: id}
		return err
	}
	var elapsed = time.Now().Sub(begin).Seconds() * 1000
	scheduler.resultChan <- SchedulerResult{ID: id}
	log.Printf("<scheduler-%s> volume '%s' shrank in %.3f milliseconds",
		scheduler.name, path, elapsed)
	return nil
}

func (scheduler *IOScheduler) handleCreateSnapshotTask(group, snapshot string, targets []snapshotTarget, respChan chan error) (err error) {
	defer func(e error) {
		var event = schedulerEvent{Type: schedulerEventCreateSnapshotCompleted, Group: group, Snapshot: snapshot, ErrorChan: respChan}
		event.Error = e
		scheduler.eventChan <- event
	}(err)
	for _, target := range targets {
		current, backing := target.Current, target.Backing
		if err = os.Rename(current, backing); err != nil {
			return err
		}
		var cmd = exec.Command("qemu-img", "create", "-f", "qcow2", "-b", backing, current)
		if err = cmd.Run(); err != nil {
			return err
		}
		log.Printf("<scheduler-%s> '%s' created on '%s'",
			scheduler.name, current, backing)
	}
	log.Printf("<scheduler-%s> %d files created with snapshot '%s.%s'",
		scheduler.name, len(targets), group, snapshot)
	return nil
}

func (scheduler *IOScheduler) handleRestoreSnapshotTask(group, snapshot string, targets []snapshotTarget, respChan chan error) (err error) {
	defer func(e error) {
		var event = schedulerEvent{Type: schedulerEventRestoreSnapshotCompleted, Group: group, Snapshot: snapshot, ErrorChan: respChan}
		event.Error = e
		scheduler.eventChan <- event
	}(err)
	for _, target := range targets {
		current, backing := target.Current, target.Backing
		if _, err = os.Stat(backing); os.IsNotExist(err) {
			err = fmt.Errorf("invalid backing path '%s'", backing)
			return err
		}
		if err = os.Remove(current); err != nil {
			return err
		}
		var cmd = exec.Command("qemu-img", "create", "-f", "qcow2", "-b", backing, current)
		if err = cmd.Run(); err != nil {
			return err
		}
		log.Printf("<scheduler-%s> '%s' reverted to '%s'",
			scheduler.name, current, backing)
	}
	log.Printf("<scheduler-%s> %d files restored with snapshot '%s.%s'",
		scheduler.name, len(targets), group, snapshot)
	return nil
}

func (scheduler *IOScheduler) handleDeleteSnapshotTask(group, snapshot string, targets []snapshotTarget, respChan chan error) (err error) {
	defer func(e error) {
		var event = schedulerEvent{Type: schedulerEventDeleteSnapshotCompleted, Group: group, Snapshot: snapshot, ErrorChan: respChan}
		event.Error = e
		scheduler.eventChan <- event
	}(err)
	for _, target := range targets {
		file := target.Backing
		if err = os.Remove(file); err != nil {
			log.Printf("<scheduler-%s> warning: delete snapshot file '%s' fail: %s",
				scheduler.name, file, err.Error())
		} else {
			log.Printf("<scheduler-%s> snapshot file '%s' deleted",
				scheduler.name, file)
		}
	}
	log.Printf("<scheduler-%s> %d files deleted with snapshot '%s.%s'",
		scheduler.name, len(targets), group, snapshot)
	return nil
}

func (scheduler *IOScheduler) generateCheckSum(target string) (sum string, err error) {
	file, err := os.Open(target)
	if err != nil {
		return
	}
	var checkBuffer = make([]byte, 4<<20) //4M buffer
	var hash = sha1.New()
	var begin = time.Now()
	bytes, err := io.CopyBuffer(hash, file, checkBuffer)
	if err != nil {
		file.Close()
		return
	}
	var elapsed = (time.Now().Sub(begin)) / time.Millisecond
	sum = hex.EncodeToString(hash.Sum(nil))
	log.Printf("<scheduler-%s> compute hash '%s' in %d millisecond(s) for %d bytes",
		scheduler.name, sum, elapsed, bytes)
	file.Close()
	return
}

func (scheduler *IOScheduler) apiPath(path string) string {
	return fmt.Sprintf("%s/v%d%s", APIRoot, APIVersion, path)
}

func bytesToString(sizeInBytes uint64) string {
	const (
		KB = 1 << 10
		MB = KB << 10
		GB = MB << 10
		TB = GB << 10
	)
	var value = float64(sizeInBytes)
	var unit string
	if value < KB {
		return fmt.Sprintf("%d Bytes", sizeInBytes)
	} else if value < MB {
		unit = "KB"
		value = value / KB
	} else if value < GB {
		unit = "MB"
		value = value / MB
	} else if value < TB {
		unit = "GB"
		value = value / GB
	} else {
		unit = "TB"
		value = value / TB
	}
	if value == math.Round(value) {
		//integer
		return fmt.Sprintf("%d %s", int(value), unit)
	} else {
		return fmt.Sprintf("%.02f %s", value, unit)
	}
}
