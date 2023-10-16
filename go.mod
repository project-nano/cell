module github.com/project-nano/cell

go 1.19

replace (
	github.com/project-nano/cell/service => ./src/service
	github.com/project-nano/cell/task => ./src/task
	github.com/project-nano/framework => ../framework
)

require (
	github.com/libvirt/libvirt-go v7.4.0+incompatible
	github.com/project-nano/cell/service v0.0.0-00010101000000-000000000000
	github.com/project-nano/cell/task v0.0.0-00010101000000-000000000000
	github.com/project-nano/framework v1.0.9
	github.com/project-nano/sonar v0.0.0-20190628085230-df7942628d6f
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/vishvananda/netlink v1.1.0
)

require (
	github.com/amoghe/go-crypt v0.0.0-20220222110647-20eada5f5964 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/klauspost/reedsolomon v1.11.8 // indirect
	github.com/krolaw/dhcp4 v0.0.0-20190909130307-a50d88189771 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sevlyar/go-daemon v0.1.6 // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
	github.com/templexxx/xor v0.0.0-20191217153810-f85b25db303b // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/xtaci/kcp-go v5.4.20+incompatible // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	golang.org/x/crypto v0.13.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)
