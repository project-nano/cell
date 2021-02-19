# Change Log

## [1.3.1] - 2021-02-19

### Added

- Set auto start

## [1.3.0] - 2020-10-29

### Added

- Allocate instance address using Cloud-Init
- Manage security policy of instance
- Create instances with security policy
 
### Changed

- Return an explicit error when resize disk fail after creating a new image
- Return an explicit error when resize volume fail
- Optimize output of image size in IO Scheduler 

## [1.2.0] - 2020-04-11 

### Added

- Query/Change storage path
- Create guest using template, use vga as the default video driver for better resolution 
- Reset monitor secret

### Changed

- Collect disk space via storage paths 

### Fixed

- Huge/Wrong available memory/disk number when no qga available in guest

## [1.1.1] - 2020-01-01 

### Changed

- Reset system before initialization change from error to a warning
- Reduce log for DHCP warning
- Network detect interval change to two minutes after established some IP
- Add CreateTime/MAC address to instance

## [1.1.0] - 2019-11-07

### Added 

- Add go mod

### Changed

- Call core API via prefix '/api/v1/'
- Change "/media_image_files/:id" to "/media_images/:id/file/"
- Change "/disk_image_files/:id" to "/disk_images/:id/file/"

## [1.0.0] - 2019-7-14

### Added

- Set threshold of CPU/Disk IO/Network

### Changed

- Generate module name base on br0

- Move to "github.com/project-nano"

## [0.8.3] - 2019-4-22

### Fixed

- Read interface fail due to script code in ifcfg

## [0.8.2] - 2019-04-04

### Fixed

- Enable Cloud-Init after resetting system image

## [0.8.1] - 2019-02-15

### Added

- Rename guest

### Changed

- Migrate bridge configure from interface

- Adapt to new runnable implement

## [0.7.1] - 2018-11-27

### Added

- "legacy" option of system version

- Reset guest system

- Sync storage option when compute pool available

## [0.6.1] - 2018-11-27

### Added

- Support assigned network address of instance

- Enable distributed DHCP service for MAC bound

- Configure template for different OS version

- Optimize mouse position using tablet input

- Disable DHCP service in default network when startup

## [0.5.1] - 2018-10-31

### Added

- Attach/Detach instances

- Add 'qcow2' suffix to volume/snapshot files

- Set hostname when using CloudInit module

- Check the default route when startup

### Changed

- Randomize allocation of the monitor port

- Determine system/admin name of the guest by version when creating an instance

## [0.4.2] - 2018-10-11

### Added

- Synchronize allocated network ports to instance configures

### Fixed

- Compatible with local storage config file of the previous version

## [0.4.1] - 2018-9-30

### Added

- Support NFS storage pool

- Report share storage(NFS) mount status

- Support volume/snapshot save on shared storage

- Create instance metadata when using shared storage

- Automount shared storage when cell start or added to compute pool

### Fixed

- Snapshot files left when delete volumes

- Try recover stub service when stop module

## [0.3.1] - 2018-8-29

### Added

- Snapshot management: create/delete/restore/get/query

- Storage create time of guest

- Inject/eject media in running instance

- Lock volumes when running disk operates

### Fixed

- Get instance status sync status without notify core

## [0.2.3] - 2018-8-14

### Added

- Support initialize guest after created using Cloud-Init in NoCloudMode

- Enable guest system version/modules configure

- Enable change admin password/create new admin/auto resize&mount disk when ci module enabled(cloud-init cloud-utils required in guest)

### Modified

- Add listen port TCP: {cellIP}:25469 for Cloud-Init initiator

## [0.2.2] - 2018-8-6

### Modified

- Enable KVM instead of TCG of QEMU, boost performance when VT-x/AMD-v enabled

- Using the IDE system disk if the system of an instance is "windows". 

- Don't save NetworkAddress of Instance

- Avoid response channel block after timeout event invoked

- Fixed: panic when try to notify the Resize/Shrink tasks

## [0.2.1] - 2018-7-29

### Added

- Modify Cores/Memory/Disk Size

- Shrink guest volume

- Set/Get user password

- Add "system" property in guest

## [0.1.4] - 2018-7-24

### Modified

- Resize guest disk when clone finished

- Compute instance CPU usage properly

- Get IP address for started instance

- Notify Core module when instance ip detected

- Fixed: Internal instance address not send to Core module

## [0.1.3] - 2018-7-19
   
### modified
   
- fix instance MAC not properly generated
 

## [0.1.2] - 2018-7-18

### modified

- add version output on the console

- add qemu-agent channel in guest

- fix instance memory usage monitor

- try to reconnect when core disconnected

- gracefully disconnect when module stop

