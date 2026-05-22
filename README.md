# UBAX-Pilot

UBAX-Pilot 是一个跨平台（Windows/Linux）的日志采集守护进程，基于 Go 语言实现。

## 架构

- **Control & Orchestration Layer**: 进程管理、系统服务适配、资源监控
- **Communication & Config Layer**: 远程配置、配置渲染、心跳上报
- **Data Core Layer**: 数据采集与边缘计算（基于 Vector）
- **Reliability Layer**: 本地缓冲、熔断限流

## 目录结构

```
cmd/ubax-pilot/     # 主入口
internal/
  control/          # 控制与编排层
  comm/             # 通信与配置层
  datacore/         # 数据采集内核
  reliability/      # 可靠性保障层
pkg/
  config/           # 共享配置
  logger/           # 日志工具
```
