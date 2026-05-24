# UBAX-Pilot

UBAX-Pilot 是一个跨平台（Windows/Linux）的采集代理守护进程，基于 Go 语言实现。负责管理 Vector 采集器的生命周期，包括进程监控、自动重启、资源配置、远程配置下发等。

## 前置要求

安装 Vector：
```powershell
Invoke-WebRequest https://packages.timber.io/vector/0.55.0/vector-x64.msi -OutFile vector-0.55.0-x64.msi
msiexec /i vector-0.55.0-x64.msi
```

## 功能

- **进程管理**：启动、监控、崩溃自动重启 Vector 进程
- **资源监控**：内存超限自动重启（连续 3 次检测触发）
- **远程配置**：周期性拉取 + 服务端主动推送（HTTP 长轮询）
- **配置渲染**：将服务端规则翻译为 Vector YAML 配置并触发热重载
- **心跳上报**：定期上报版本、主机名、操作系统、Vector 运行状态
- **远程命令**：支持服务端下发 `restart`/`stop`/`reload`/`upgrade` 命令
- **系统服务**：支持安装为 Windows Service 或 Linux Systemd 服务

## 编译

```bash
go build -o ubax-pilot.exe ./cmd/ubax-pilot/
```

## 使用

```bash
# 显示版本号
ubax-pilot --version
ubax-pilot -v

# 运行（自动加载默认路径配置）
ubax-pilot

# 指定配置文件
ubax-pilot --config /path/to/config.yaml
ubax-pilot -c /path/to/config.yaml

# 安装为系统服务
ubax-pilot --install
ubax-pilot -i

# 卸载系统服务
ubax-pilot --uninstall
ubax-pilot -u
```

## 配置

配置文件默认路径：
- Windows: `C:\ProgramData\ubax-pilot\config\config.yaml`
- Linux: `/etc/ubax-pilot/config.yaml`

示例配置：
```yaml
server_endpoint: "http://localhost:9090"
heartbeat_interval_seconds: 30
vector_bin_path: "C:\\Program Files\\Vector\\bin\\vector.exe"
vector_conf_path: "C:\\Program Files\\Vector\\config\\vector.yaml"
max_memory_mb: 512
```

版本号等内置信息从 `pkg/config/built_in.yaml` 读取。

## 架构

```
┌─────────────────────────────────────────────────┐
│                  UBAX-Pilot                      │
│                                                  │
│  ┌──────────────┐    ┌────────────────────────┐  │
│  │  进程管理     │    │  远程配置拉取/推送      │  │
│  │  ProcessMgr  │◄──►│  ConfigRenderer        │  │
│  └──────┬───────┘    └────────────┬───────────┘  │
│         │                         │              │
│  ┌──────▼───────┐    ┌────────────▼───────────┐  │
│  │  资源监控     │    │  心跳上报               │  │
│  │  ResourceMon │    │  HeartbeatReporter     │  │
│  └──────────────┘    └────────────────────────┘  │
└─────────────────────────────────────────────────┘
         │
         ▼
    ┌─────────┐
    │ Vector  │  (数据采集 → 直接上传)
    └─────────┘
```

## 目录结构

```
cmd/ubax-pilot/         # 主入口
internal/
  control/              # 控制与编排层
    process_manager.go  #   Vector 进程生命周期管理
    service_adapter.go  #   服务适配器基础
    service_linux.go    #   Linux Systemd 实现
    service_windows.go  #   Windows Service 实现
  comm/                 # 通信与配置层
    config_renderer.go  #   Vector 配置渲染
    heartbeat.go        #   心跳上报
    remote_config.go    #   远程配置轮询
    server_push.go      #   服务端推送客户端
pkg/
  config/               # 配置管理
    built_in.yaml       #   内置配置（版本号等）
    config.go           #   运行时配置
  logger/               # 日志工具
```
