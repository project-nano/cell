module github.com/project-nano/cell

go 1.13

replace (
	github.com/project-nano/cell/service => ./src/service
	github.com/project-nano/cell/task => ./src/task
)

require (
	github.com/amoghe/go-crypt v0.0.0-20151031192136-f85b640b0eef // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/klauspost/reedsolomon v1.9.3 // indirect
	github.com/krolaw/dhcp4 v0.0.0-20190909130307-a50d88189771 // indirect
	github.com/libvirt/libvirt-go v5.8.0+incompatible
	github.com/pkg/errors v0.8.1 // indirect
	github.com/project-nano/cell/service v0.0.0-00010101000000-000000000000
	github.com/project-nano/cell/task v0.0.0-00010101000000-000000000000
	github.com/project-nano/framework v1.0.1
	github.com/project-nano/sonar v0.0.0-20190628085230-df7942628d6f
	github.com/sevlyar/go-daemon v0.1.5 // indirect
	github.com/shirou/gopsutil v2.19.10+incompatible
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
	github.com/templexxx/xor v0.0.0-20181023030647-4e92f724b73b // indirect
	github.com/tjfoc/gmsm v1.0.1 // indirect
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	github.com/xtaci/kcp-go v5.4.19+incompatible // indirect
	golang.org/x/crypto v0.0.0-20191106202628-ed6320f186d4 // indirect
	golang.org/x/net v0.0.0-20191105084925-a882066a44e0 // indirect
	golang.org/x/sys v0.0.0-20191105231009-c1f44814a5cd // indirect
)
