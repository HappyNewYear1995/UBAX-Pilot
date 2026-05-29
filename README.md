# UBAX-Pilot

UBAX-Pilot 是一个跨平台（Windows/Linux）的采集代理守护进程，基于 Go 语言实现。负责管理 Vector
采集器的生命周期，包括进程监控、自动重启、资源监控、远程配置下发等。

> **设计理念**：ubax-pilot 仅负责监控、维护和控制，数据采集和上传完全由 Vector 自行处理。

## 功能

| 模块       | 能力                                            |
|----------|-----------------------------------------------|
| **进程管理** | 启动、监控、崩溃自动重启（指数退避策略）                          |
| **资源监控** | 内存超限自动重启（连续 3 次检测触发）                          |
| **远程配置** | 服务端主动推送（SSE 长连接），自动写入 Vector 配置文件             |
| **心跳上报** | 定期上报 UUID、版本、主机名、操作系统、Vector 运行状态             |
| **远程命令** | 支持服务端下发 `restart` / `stop` 命令，控制 Vector 重启和关闭 |
| **系统服务** | 支持安装为 Windows Service 或 Linux Systemd 服务      |
| **唯一标识** | 首次运行自动生成 UUID 并持久化，用于服务端唯一识别                  |

## 前置要求

### 安装 Vector

Windows 示例：

```powershell
Invoke-WebRequest https://packages.timber.io/vector/0.55.0/vector-x64.msi -OutFile vector-0.55.0-x64.msi

msiexec /i vector-0.55.0-x64.msi
```

Linux 示例：

```bash
curl --proto '=https' --tlsv1.2 -sSfL https://sh.vector.dev | bash
```

可参考 [Vector 官网](https://vector.dev/ "Vector 官网")

## 编译

### Windows (x64)：

```bash
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o ubax-pilot.exe ./cmd/ubax-pilot/
```

### Linux (x64)：

```bash
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o ubax-pilot ./cmd/ubax-pilot/
```

## 使用

```bash
# 显示版本号
ubax-pilot --version   # 或 -v

# 安装为系统服务（推荐）
ubax-pilot --install   # 或 -i

# 卸载系统服务
ubax-pilot --uninstall   # 或 -u

# 运行（自动加载默认路径配置）
ubax-pilot

# 指定配置文件
ubax-pilot --config /path/to/config.yaml   # 或 -c
```

## 配置

配置文件默认路径：

- Windows: `C:\ProgramData\ubax-pilot\config\config.yaml`
- Linux: `/etc/ubax-pilot/config.yaml`

首次运行时自动生成默认配置，包含唯一 `agent_uuid`：

```yaml
agent_uuid: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
server_endpoint: "http://localhost:9090"
heartbeat_interval_seconds: 30
vector_bin_path: "C:\\Program Files\\Vector\\bin\\vector.exe"
vector_conf_path: "C:\\Program Files\\Vector\\config\\vector.yaml"
max_memory_mb: 512
```

| 配置项                          | 说明                                                                  |
|------------------------------|---------------------------------------------------------------------|
| `agent_uuid`                 | 唯一标识，首次运行自动生成，不再修改                                                  |
| `server_endpoint`            | 服务端地址（ [UBA-X](https://github.com/HappyNewYear1995/UBA-X "UBA-X") ） |
| `heartbeat_interval_seconds` | 心跳上报间隔（秒）                                                           |
| `vector_bin_path`            | Vector 可执行文件路径                                                      |
| `vector_conf_path`           | Vector 配置文件路径                                                       |
| `max_memory_mb`              | 内存上限（MB），超限自动重启                                                     |

## 架构

```
┌──────────────────────────────────────────────────────────────────────┐
│                         UBAX 服务端                                   │
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐    │
│  │  配置管理     │    │  设备管理     │    │  推送服务             │    │
│  │ Config Mgr  │    │ Device Mgr  │    │ SSE Push Server      │    │
│  └──────┬───────┘    └──────┬───────┘    └──────────┬───────────┘    │
│         │                   │                       │                │
│         └───────────────────┼───────────────────────┘                │
│                             │                                        │
│  ┌──────────────────────────▼───────────────────────────────────┐    │
│  │                    API 网关                                    │    │
│  │  /pilot/agent/heartbeat  (POST)  ← 心跳接收                   │    │
│  │  /pilot/agent/content    (SSE)   ← 配置/命令推送              │    │
│  └──────────────────────────────────────────────────────────────┘    │
└──────────────────────────────┬───────────────────────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              │  SSE 推送       │   HTTP POST    │
              │  (配置/命令)    │   (心跳上报)    │
              ▼                │                ▲
┌─────────────────────────────────────────────────────────────────────┐
│                         UBAX-Pilot 客户端                            │
│                                                                      │
│  ┌────────────────────────────┐    ┌────────────────────────────┐    │
│  │  SSE 推送客户端             │    │  心跳上报器                  │    │
│  │  ServerPushClient          │    │  HeartbeatReporter         │    │
│  │  • 接收配置推送             │    │  • 定期上报状态             │    │
│  │  • 接收远程命令             │    │  • 携带 UUID/版本/状态      │    │
│  └────────────┬───────────────┘    └────────────┬───────────────┘    │
│               │                                 │                    │
│               ▼                                 │                    │
│  ┌────────────────────────────┐                 │                    │
│  │  配置渲染器                 │                 │                    │
│  │  ConfigRenderer            │                 │                    │
│  │  • 渲染 Vector YAML 配置   │                 │                    │
│  └────────────┬───────────────┘                 │                    │
│               │                                 │                    │
│  ┌────────────▼───────────────┐    ┌────────────▼───────────────┐    │
│  │  进程管理器                 │    │  资源监控器                 │    │
│  │  ProcessManager            │    │  ResourceMonitor           │    │
│  │  • 启动/停止/重启 Vector   │◄──►│  • 内存超限检测             │    │
│  │  • 崩溃自动重启(指数退避)   │    │  • 连续 3 次触发重启        │    │
│  └────────────┬───────────────┘    └────────────────────────────┘    │
│               │                                                       │
└───────────────┼───────────────────────────────────────────────────────┘
                ▼
    ┌────────────────────────┐
    │        Vector          │
    │  • 数据采集             │
    │  • 数据上传             │
    │  • --watch-config 自动重载 │
    └────────────────────────┘
```

## 通信流程

```
UBAX 服务端                              UBAX-Pilot 客户端
    │                                         │
    │  ←── SSE 长连接 ──→                     │
    │  (配置推送 / 命令下发)                   │
    │                                         │
    │  ←── HTTP POST 心跳 ──                  │
    │  (每 30s 上报状态)                       │
    │                                         │
    │  ──→ Vector 数据上传 ──→  目标服务       │
    │     (不经过 ubax-pilot)                  │
```

## 服务端对接

ubax-pilot 通过以下方式与服务端通信：

| 方向   | 协议                                 | 说明           |
|------|------------------------------------|--------------|
| 心跳上报 | HTTP POST `/pilot/agent/heartbeat` | 定期上报状态       |
| 配置推送 | SSE 长连接 `/pilot/agent/content`   | 服务端主动推送配置和命令 |

## 目录结构

```
cmd/ubax-pilot/              # 主入口
internal/
  control/                   # 控制与编排层
    process_manager.go       #   Vector 进程生命周期管理
    service_adapter.go       #   服务适配器基础
    service_linux.go         #   Linux Systemd 实现
    service_windows.go       #   Windows Service 实现
  comm/                      # 通信层
    config_renderer.go       #   Vector 配置渲染
    heartbeat.go             #   心跳上报
    server_push.go           #   服务端推送客户端（SSE）
pkg/
  config/                    # 配置管理
    built_in.yaml            #   内置配置（版本号等）
    config.go                #   运行时配置
    built_in.go              #   内置配置加载
  logger/                    # 日志工具
config/                      # 项目示例配置目录
```
