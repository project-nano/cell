# Nano Cell

[[版本历史/ChangeLog](CHANGELOG.md)]

[English Version](#introduce)

### 简介

Cell模块是Nano集群的资源节点，用于组成虚拟化资源池，在本地创建、管理和释放虚拟机实例。

由于涉及网络配置，建议使用专用Installer进行部署，项目最新版本请访问[此地址](https://github.com/project-nano/releases)

[项目官网](https://nanos.cloud/)

[项目全部源代码](https://github.com/project-nano)

### 编译

环境要求

- CentOS 7 x86
- Golang 1.20

```
准备依赖的framework
$git clone https://github.com/project-nano/framework.git

准备编译源代码
$git clone https://github.com/project-nano/cell.git

编译
$cd cell
$go build
```

编译成功在当前目录生成二进制文件cell

### 使用

环境要求

- CentOS 7 x86

```
执行以下指令，启动Cell模块
$./cell start

也可以使用绝对地址调用或者写入开机启动脚本，比如
$/opt/nano/cell/cell start

```

模块运行日志输出在log/cell.log文件中，用于查错和调试

**由于Cell节点依赖Core模块进行自动网络识别，所以集群工作时必须最先启动Core模块，再启动Cell节点。**

此外，除了模块启动功能，Cell还支持以下命令参数启动

| 命令名 | 说明                               |
| ------ | ---------------------------------- |
| start  | 启动服务                           |
| stop   | 停止服务                           |
| status | 检查当前服务状态                   |
| halt   | 强行中止服务（用于服务异常时重启） |



### 配置

Cell模块配置信息存放在config路径文件中，修改后需要重启模块生效

#### 域通讯配置

文件config/domain.cfg管理Cell模块的域通讯信息，域参数必须与Core模块一致才能成功识别

| 参数              | 值类型 | 默认值      | 必填 | 说明                         |
| ----------------- | ------ | ----------- | ---- | ---------------------------- |
| **domain**        | 字符串 | nano        | 是   | 通讯域名称，用于节点间识别   |
| **group_address** | 字符串 | 224.0.0.226 | 是   | 通讯域组播地址，用于服务发现 |
| **group_port**    | 整数   | 5599        | 是   | 通讯域组播端口，用于服务发现 |
| **timeout**       | 整数   | 10          |      | 交易处理超时时间，单位：秒   |

示例配置文件如下

```json
{
 "domain": "nano",
 "group_address": "224.0.0.226",
 "group_port": 5599
}
```



### 目录结构

模块主要目录和文件如下

| 目录/文件 | 说明                 |
| --------- | -------------------- |
| cell      | 模块二进制执行文件   |
| config/   | 配置文件存储目录     |
| data/     | 模块运行数据存储目录 |
| log/      | 运行日志存储目录     |

# README

### Introduce

Cell is the resource node of Nano cluster, forming a virtualized resource pool. It creates, manages, and releases local virtual machine instances.

It is recommended to use a dedicated Installer for deployment. For the latest project version, please visit [this address](https://github.com/project-nano/releases).

[Official Project Website](https://us.nanos.cloud/en/)

[Full Source Code of the Project](https://github.com/project-nano)

### Compilation

Environment requirements

- CentOS 7 x86
- Golang 1.20

```bash
Prepare the framework dependencies
$git clone https://github.com/project-nano/framework.git

Prepare the source code for compilation
$git clone https://github.com/project-nano/cell.git

Compile
$cd cell
$go build
    
```

The binary file "cell" will be generated in the current directory when success

### Usage

Environment

- CentOS 7 x86

```bash
start module
$./cell start

Alternatively, you can use an absolute address or write it into a startup script, such as:
$/opt/nano/cell/cell start
    
```

The running log is output on the file: log/core.log

**Since the Cell nodes depend on the Core module for automatic network recognition, you must start the Core module before Cell nodes.**

Core also supports the following command

| Command name | Explanation                               |
| ------------ | ----------------------------------------- |
| start        | Start service                             |
| stop         | Stop service                              |
| status       | Check current service status              |
| halt         | Force abort service when exception occurs |



### Configuration

Cell module configuration information is stored in files under the config path, and modifications require a restart of the module to take effect.

#### Domain Communication

The file `config/domain.cfg` manages the domain communication information for the Cell module.

| Parameter         | Value Type | Default Value | Required | Explanation                                                  |
| ----------------- | ---------- | ------------- | -------- | ------------------------------------------------------------ |
| **domain**        | String     | nano          | Yes      | The name of the communication domain, used for cluster identification |
| **group_address** | String     | 224.0.0.226   | Yes      | Multicast address of the communication domain, used for service discovery |
| **group_port**    | Integer    | 5599          | Yes      | Multicast port of the communication domain, used for service discovery |
| **timeout**       | Integer    | 10            |          | Transaction timeout in seconds                               |

An example configuration file is as follows:

```json
{
 "domain": "nano",
 "group_address": "224.0.0.226",
 "group_port": 5599
}
```



### Directory Structure

| Directory/File | Explanation                              |
| -------------- | ---------------------------------------- |
| cell           | The binary execution file of the module  |
| config/        | The storage directory for configurations |
| data/          | The storage directory for operation data |
| log/           | The storage directory for logs           |
