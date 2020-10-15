module github.com/project-nano/cell

go 1.13

replace (
	github.com/project-nano/cell/service => ./src/service
	github.com/project-nano/cell/task => ./src/task
	github.com/project-nano/framework => ../framework
)

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/libvirt/libvirt-go v6.1.0+incompatible
	github.com/project-nano/cell/service v0.0.0-00010101000000-000000000000
	github.com/project-nano/cell/task v0.0.0-00010101000000-000000000000
	github.com/project-nano/framework v1.0.3
	github.com/project-nano/sonar v0.0.0-20190628085230-df7942628d6f
	github.com/shirou/gopsutil v2.19.10+incompatible
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
)
