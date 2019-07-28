# README

## Overview

The Cell works on every server that needs to join the resource pool, it reports the resource usage and allocates instances when the Core request.

Start the Core module first, then the Cell module. A server node can only visible and able to host the instance when the Cell started properly.

Ensure the Cell has the identical communicate domain configuration to the Core you want to join.

Binary release found [here](<https://github.com/project-nano/releases/releases>)

See more detail for [Quick Guide](<https://nanocloud.readthedocs.io/projects/guide/en/latest/concept.html>)

Official Site: <https://nanos.cloud/en-us/>

REST API: <https://nanoen.docs.apiary.io/>

Wiki: <https://github.com/project-nano/releases/wiki/English>

## Build

Assume that the golang lib installed in the '/home/develop/go',  and source code downloaded in the path '/home/develop/nano/cell'.

Set environment variable GOPATH before compiling

```
#git clone https://github.com/project-nano/cell.git
#cd cell
#export GOPATH="/home/develop/go:/home/develop/nano/cell"
#go build -o cell -i -ldflags="-w -s"
```



## Command Line

All Nano modules provide the command-line interface, and called like :

< module name > [start | stop | status | halt]

- start: start module service, output error message when start failed, or version information.
- stop: stops the service gracefully. Releases allocated resources and notify any related modules.
- status: checks if the module is running.
- halt: terminate service immediately.

You can call the Core module both in the absolute path and relative path.

```
#cd /opt/nano/cell
#./cell start

or

#/opt/nano/cell/cell start
```

Please check the log file "log/cell.log" when encountering errors.

## Configure

All configure files stores in the path: config

### Domain Configuration

Filename: domain.cfg

See more detail about [Domain](<https://nanocloud.readthedocs.io/projects/guide/en/latest/concept.html#communicate-domain>)

| Parameter         | Description                                                  |
| ----------------- | ------------------------------------------------------------ |
| **domain**        | The name of a communication domain, like 'nano' in default, only allows characters. |
| **group_port**    | Multicast port, 5599 in default                              |
| **group_address** | Multicast address, '224.0.0.226' in default.                 |

