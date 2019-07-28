package service

import (
	"github.com/libvirt/libvirt-go"
	"encoding/xml"
	"fmt"
	"errors"
	"os/user"
	"os"
	"path/filepath"
	"log"
	"bytes"
	"os/exec"
)

type virStorageVolume struct {
	XMLName    xml.Name               `xml:"volume"`
	Type       string                 `xml:"type,attr"`
	Name       string                 `xml:"name"`
	Key        string                 `xml:"key,omitempty"`
	Allocation uint64                 `xml:"allocation,omitempty"`
	Capacity   uint64                 `xml:"capacity,omitempty"`
	Target     virStorageVolumeTarget `xml:"target"`
}

type virStorageVolumeTarget struct {
	Path        string                `xml:"path,omitempty"`
	Format      virVolumeTargetFormat `xml:"format"`
	Permissions virTargetPermissions  `xml:"permissions,omitempty"`
	Compat      string                `xml:"compat,omitempty"`
}

type virVolumeTargetFormat struct {
	Type string `xml:"type,attr"`
}

type virTargetPermissions struct {
	Owner string `xml:"owner"`
	Group string `xml:"group"`
	Mode  string `xml:"mode,omitempty"`
	Label string `xml:"label,omitempty"`
}

type virPoolTargetDefine struct {
	Path        string               `xml:"path,omitempty"`
	Permissions virTargetPermissions `xml:"permissions,omitempty"`
}

type virPoolSourceHost struct {
	Name string `xml:"name,attr"`
}

type virPoolSourceDIR struct {
	Path string `xml:"path,attr"`
}

type virPoolSourceFormat struct {
	Type string `xml:"type,attr"`
}

type virPoolSourceDefine struct {
	Host   virPoolSourceHost   `xml:"host"`
	Dir    virPoolSourceDIR    `xml:"dir"`
	Format virPoolSourceFormat `xml:"format"`
}

type virStoragePoolDefine struct {
	XMLName xml.Name             `xml:"pool"`
	Type    string               `xml:"type,attr"`
	Name    string               `xml:"name"`
	UUID    string               `xml:"uuid.omitempty"`
	Source  *virPoolSourceDefine `xml:"source,omitempty"`
	Target  *virPoolTargetDefine `xml:"target,omitempty"`
}

type StorageUtility struct {
	poolsPath    string
	currentUser  *user.User
	innerConnect *libvirt.Connect
}

const (
	StoragePoolTypeNameLocal   = "dir"
	StoragePoolTypeNameNetFile = "netfs"
)

const (
	DefaultVolumeFormat   = "qcow2"
	DefaultVolumeType     = "file"
	DefaultQCOW2Compat    = "1.1"
	DefaultVolumeFileMode = "0666"
	SharePoolPathName     = "pools"
	DefaultPathPerm       = 0744
)

const (
	StoragePoolFormatNFS = "nfs"
)

func CreateStorageUtility(connect *libvirt.Connect) (util *StorageUtility, err error) {
	util = &StorageUtility{}
	util.currentUser, err = user.Current()
	if err != nil{
		return nil, err
	}
	//build pools path
	currentPath, err := os.Getwd()
	if err != nil{
		return
	}
	util.poolsPath = filepath.Join(filepath.Dir(currentPath), SharePoolPathName)
	util.innerConnect = connect
	return
}

func (util *StorageUtility) HasPool(name string) bool {
	//todo: check storage type
	if _, err := util.innerConnect.LookupStoragePoolByName(name); err != nil{
		return false
	}
	return true
}

func (util *StorageUtility) RefreshPool(poolName string) (err error){
	virPool, err := util.innerConnect.LookupStoragePoolByName(poolName)
	if err != nil{
		return err
	}
	return virPool.Refresh(0)
}

func (util *StorageUtility) GetPool(name string) (pool StoragePool, err error) {

	virPool, err := util.innerConnect.LookupStoragePoolByName(name)
	if err != nil{
		return pool, err
	}
	description, err := virPool.GetXMLDesc(0)
	if err != nil{
		return pool, err
	}
	var poolDefine virStoragePoolDefine
	if err = xml.Unmarshal([]byte(description), &poolDefine);err != nil{
		return pool, err
	}
	pool.Name = poolDefine.Name
	pool.ID = poolDefine.UUID
	switch poolDefine.Type{
	case StoragePoolTypeNameLocal:
		pool.Mode = StoragePoolModeLocal
		break
	default:
		return pool, fmt.Errorf("unsupported pool type '%s'", poolDefine.Type)
	}
	if poolDefine.Target == nil{
		return pool, errors.New("can not find target path")
	}
	pool.Target = poolDefine.Target.Path
	err = pool.UpdateCapacity(virPool)
	return pool, err
}

func (util *StorageUtility) EnableNFSPools() (err error){
	{
		var output bytes.Buffer
		var cmd = exec.Command("setsebool", "-P", "virt_use_nfs", "1")
		cmd.Stderr = &output
		if err = cmd.Run();err != nil{
			err = fmt.Errorf("enable virt on NFS fail: %s, output: %s", err.Error(), output.String())
			return
		}
	}
	if _, err = os.Stat(util.poolsPath);os.IsNotExist(err){
		if err = os.MkdirAll(util.poolsPath, DefaultPathPerm);err != nil{
			return
		}
		log.Printf("<storage> pools path '%s' created", util.poolsPath)
		if err = util.setSelinuxLabel(util.poolsPath); err != nil{
			return
		}
		if err = util.restoreSelinuxLabel(util.poolsPath); err != nil{
			return
		}
		log.Printf("<storage> selinux label enabled in '%s'", util.poolsPath)
	}
	return nil
}

func (util *StorageUtility) CreateNFSPool(name, host, target string) (pool StoragePool, err error) {
	var poolPath string
	{
		//pool path
		poolPath = filepath.Join(util.poolsPath, name)
		if _, err = os.Stat(poolPath);os.IsNotExist(err){
			if err = os.MkdirAll(poolPath, DefaultPathPerm); err != nil{
				err = fmt.Errorf("create pool path '%s' fail: %s", poolPath, err.Error())
				return
			}
			log.Printf("<storage> pool path '%s' created", poolPath)
			if err = util.restoreSelinuxLabel(poolPath);err != nil{
				return
			}
			log.Printf("<storage> selabel of path '%s' restored", poolPath)
		}
	}
	return util.buildVirNFSPool(name, host, target, poolPath)
}

func (util *StorageUtility) DeleteNFSPool(poolName string) (err error){
	if err = util.deletePool(poolName); err != nil{
		return
	}
	var poolPath = filepath.Join(util.poolsPath, poolName)
	if _, err = os.Stat(poolPath);!os.IsNotExist(err){
		//exists
		return os.RemoveAll(poolPath)
	}
	return nil
}

func (util *StorageUtility) ChangeNFSPool(name, host, target string) (pool StoragePool, err error) {
	{
		//remove previous pool
		virPool, err := util.innerConnect.LookupStoragePoolByName(name)
		if err != nil{
			return pool, err
		}
		isActive, err := virPool.IsActive()
		if err != nil{
			return  pool, err
		}
		if isActive{
			//stop first
			if err = virPool.Destroy(); err != nil{
				return  pool, err
			}
			log.Printf("<storage> nfs pool '%s' stopped", name)
		}
		if err = virPool.Undefine();err != nil{
			return  pool, err
		}
		log.Printf("<storage> previous nfs pool '%s' undefined", name)
	}
	var poolPath = filepath.Join(util.poolsPath, name)
	return util.buildVirNFSPool(name, host, target, poolPath)
}

func (util *StorageUtility) IsNFSPoolMounted(name string) (mounted bool, err error){
	//possible long time running
	log.Printf("<storage> try check nfs pool '%s'...", name)
	virPool, err := util.innerConnect.LookupStoragePoolByName(name)
	if err != nil{
		return false, err
	}
	activated, err := virPool.IsActive()
	if err != nil{
		return false, err
	}
	if !activated {
		err = fmt.Errorf("storage pool '%s' not active, please check nfs server", name)
		return
	}
	var tagFile = fmt.Sprintf("%s.ready", name)
	var mountPath = filepath.Join(util.poolsPath, name)
	if _, err = os.Stat(mountPath);os.IsNotExist(err){
		err = fmt.Errorf("mount path '%s' not created for pool '%s'", mountPath, name)
		return false, err
	}
	var tagPath = filepath.Join(util.poolsPath, name, tagFile)
	if _, err = os.Stat(tagPath);os.IsNotExist(err){
		//no tag available
		err = fmt.Errorf("no tag '%s' available", tagPath)
		return
	}
	return true, nil
}

func (util *StorageUtility) buildVirNFSPool(name, host, target, path string) (pool StoragePool, err error){
	//define&start
	var define = virStoragePoolDefine{}
	define.Name = name
	define.Type = StoragePoolTypeNameNetFile

	define.Source = &virPoolSourceDefine{}
	define.Source.Host.Name = host
	define.Source.Dir.Path = target
	define.Source.Format.Type = StoragePoolFormatNFS

	define.Target = &virPoolTargetDefine{Path:path}
	define.Target.Permissions.Group = util.currentUser.Gid
	define.Target.Permissions.Owner = util.currentUser.Uid

	xml, err := xml.MarshalIndent(define, "", " ")
	if err != nil{
		return pool, err
	}
	//define
	virPool, err := util.innerConnect.StoragePoolDefineXML(string(xml), 0)
	if err != nil{
		return pool, err
	}
	if pool.ID, err = virPool.GetUUIDString(); err != nil{
		virPool.Undefine()
		return pool, err
	}
	if err = virPool.SetAutostart(true);err != nil{
		virPool.Undefine()
		return pool, err
	}
	if err = virPool.Create(libvirt.STORAGE_POOL_CREATE_WITH_BUILD); err != nil{
		virPool.Undefine()
		return pool, err
	}
	if err = virPool.Refresh(0); err != nil{
		virPool.Undefine()
		return pool, err
	}
	var tagFile = filepath.Join(util.poolsPath, name, fmt.Sprintf("%s.ready", name))
	if _, err = os.Stat(tagFile); os.IsNotExist(err){
		if _, err = os.Create(tagFile); err != nil{
			fmt.Errorf("generate tag '%s' fail: %s", tagFile, err.Error())
			return 
		}
	}
	pool.Name = name
	pool.SourceHost = host
	pool.SourceTarget = target
	pool.Mode = StoragePoolModeNFS
	pool.Target = path
	err = pool.UpdateCapacity(virPool)
	return pool, err
}

func (util *StorageUtility) setSelinuxLabel(path string) (err error){
	const (
		SELinuxUser = "system_u"
		SELinuxType = "virt_image_t"
	)
	var matchrule = fmt.Sprintf("%s(/.*)?", path)
	var output bytes.Buffer
	var cmd = exec.Command("semanage", "fcontext", "-a", "-s", SELinuxUser, "-t", SELinuxType, matchrule)
	cmd.Stderr = &output
	if err = cmd.Run();err != nil{
		err = fmt.Errorf("enable selinux in path '%s' fail: %s, output: %s", path, err.Error(), output.String())
		return
	}
	return nil
}

func (util *StorageUtility) restoreSelinuxLabel(path string) (err error) {
	var output bytes.Buffer
	var cmd = exec.Command("restorecon", "-F", "-R", path)
	cmd.Stderr = &output
	if err = cmd.Run();err != nil{
		err = fmt.Errorf("restore selinux in path '%s' fail: %s, output: %s", path, err.Error(), output.String())
		return
	}
	return nil
}

func (util *StorageUtility) CreateLocalPool(name, path string) (pool StoragePool, err error) {
	var define = virStoragePoolDefine{}
	define.Name = name
	define.Type = StoragePoolTypeNameLocal
	define.Target = &virPoolTargetDefine{Path:path}
	define.Target.Permissions.Group = util.currentUser.Gid
	define.Target.Permissions.Owner = util.currentUser.Uid
	//define.Target.Permissions.Label = VirtLabel

	xml, err := xml.MarshalIndent(define, "", " ")
	if err != nil{
		return pool, err
	}
	//define
	virPool, err := util.innerConnect.StoragePoolDefineXML(string(xml), 0)
	if err != nil{
		return pool, err
	}
	if pool.ID, err = virPool.GetUUIDString(); err != nil{
		virPool.Undefine()
		return pool, err
	}
	if err = virPool.SetAutostart(true);err != nil{
		virPool.Undefine()
		return pool, err
	}
	if err = virPool.Create(libvirt.STORAGE_POOL_CREATE_WITH_BUILD); err != nil{
		virPool.Undefine()
		return pool, err
	}
	pool.Name = name
	pool.Mode = StoragePoolModeLocal
	pool.Target = path
	err = pool.UpdateCapacity(virPool)
	return pool, err
}

func (util *StorageUtility) StartPool(name string) (error) {
	virPool, err := util.innerConnect.LookupStoragePoolByName(name)
	if err != nil{
		return err
	}
	activated, err := virPool.IsActive()
	if err != nil{
		return err
	}
	if !activated {
		return virPool.Create(libvirt.STORAGE_POOL_CREATE_WITH_BUILD)
	}
	return nil
}

func (util *StorageUtility) StopPool(name string) (error) {
	virPool, err := util.innerConnect.LookupStoragePoolByName(name)
	if err != nil{
		return err
	}
	activated, err := virPool.IsActive()
	if err != nil{
		return err
	}
	if activated {
		return nil
	}
	return virPool.Destroy()
}


func (util *StorageUtility) deletePool(name string) (error) {
	virPool, err := util.innerConnect.LookupStoragePoolByName(name)
	if err != nil{
		return err
	}
	activated, err := virPool.IsActive()
	if err != nil{
		return err
	}
	if activated {
		if err = virPool.Destroy(); err != nil{
			return err
		}
	}
	return virPool.Undefine()
}

func (util *StorageUtility) CreateVolumes(pool string, count int, volNames []string, volSizes []uint64) ([]StorageVolume, error) {
	virPool, err := util.innerConnect.LookupStoragePoolByName(pool)
	if err != nil{
		return nil, err
	}
	if len(volNames) != count{
		return nil, fmt.Errorf("unmatched name params count %d / %d", len(volNames), count)
	}
	if len(volSizes) != count{
		return nil, fmt.Errorf("unmatched size params count %d / %d", len(volSizes), count)
	}

	var volFormat = DefaultVolumeFormat

	var result []StorageVolume
	for i := 0; i < count;i++{
		var volName = volNames[i]
		var volSize = volSizes[i]
		var define = virStorageVolume{Name:volName, Capacity:volSize, Type:DefaultVolumeType}
		define.Target.Format = virVolumeTargetFormat{Type:volFormat}
		define.Target.Compat = DefaultQCOW2Compat
		define.Target.Permissions.Group = util.currentUser.Gid
		define.Target.Permissions.Owner = util.currentUser.Uid
		define.Target.Permissions.Mode = DefaultVolumeFileMode
		//define.Target.Permissions.Label = VirtLabel

		data, err := xml.MarshalIndent(define, "", " ")
		if err != nil{
			return nil, err
		}
		virVol, err := virPool.StorageVolCreateXML(string(data), libvirt.STORAGE_VOL_CREATE_PREALLOC_METADATA)
		if err != nil{
			return nil, err
		}
		desc, err := virVol.GetXMLDesc(0)
		if err != nil{
			return nil, err
		}
		if err = xml.Unmarshal([]byte(desc), &define); err != nil{
			return nil, err
		}

		var volume = StorageVolume{Name:volName, Format:volFormat, Key:define.Key, Path:define.Target.Path,
			Capacity:define.Capacity, Allocation:define.Allocation}
		result = append(result, volume)
	}
	return result, nil
}

func (util *StorageUtility) DeleteVolumes(pool string, volNames []string) error {
	virPool, err := util.innerConnect.LookupStoragePoolByName(pool)
	if err != nil{
		return err
	}
	for _, name := range volNames{
		vol, err := virPool.LookupStorageVolByName(name)
		if err != nil{
			return err
		}
		if err = vol.Delete(libvirt.STORAGE_VOL_DELETE_NORMAL); err != nil{
			return err
		}
	}
	return nil
}

func (util *StorageUtility) GetVolume(pool, name string) (vol StorageVolume, err error) {
	virPool, err := util.innerConnect.LookupStoragePoolByName(pool)
	if err != nil{
		return vol, err
	}
	virVol, err := virPool.LookupStorageVolByName(name)
	desc, err := virVol.GetXMLDesc(0)
	if err != nil{
		return vol, err
	}
	var define virStorageVolume
	if err = xml.Unmarshal([]byte(desc), &define); err != nil{
		return vol, err
	}

	vol = StorageVolume{Name:name, Format:define.Target.Format.Type, Key:define.Key, Path:define.Target.Path,
		Capacity:define.Capacity, Allocation:define.Allocation}
	return vol, nil
}

func (pool *StoragePool)UpdateCapacity(virPool *libvirt.StoragePool) error{
	info, err := virPool.GetInfo()
	if err != nil{
		return err
	}
	pool.Capacity = info.Capacity
	pool.Available = info.Available
	pool.Allocation = info.Allocation
	return nil
}