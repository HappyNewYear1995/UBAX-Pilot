package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ubax/ubax-pilot/internal/comm"
	"github.com/ubax/ubax-pilot/internal/control"
	"github.com/ubax/ubax-pilot/pkg/config"
	"github.com/ubax/ubax-pilot/pkg/logger"
)

var (
	configPath string
	install    bool
	uninstall  bool
	version    bool
)

func init() {
	flag.StringVar(&configPath, "config", "", "配置文件路径")
	flag.StringVar(&configPath, "c", "", "配置文件路径（简写）")
	flag.BoolVar(&install, "install", false, "安装为系统服务")
	flag.BoolVar(&install, "i", false, "安装为系统服务（简写）")
	flag.BoolVar(&uninstall, "uninstall", false, "卸载系统服务")
	flag.BoolVar(&uninstall, "u", false, "卸载系统服务（简写）")
	flag.BoolVar(&version, "version", false, "显示版本号")
	flag.BoolVar(&version, "v", false, "显示版本号（简写）")
}

func main() {
	flag.Parse()

	if version {
		fmt.Printf("UBAX-Pilot Version: %s\n", config.GetVersion())
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("加载配置失败:", err)
		os.Exit(1)
	}

	// 处理服务安装
	if install {
		adapter := control.NewServiceAdapter()
		if err := adapter.Install(); err != nil {
			logger.Error("安装服务失败:", err)
			os.Exit(1)
		}
		logger.Info("服务安装成功")
		return
	}

	if uninstall {
		adapter := control.NewServiceAdapter()
		if err := adapter.Uninstall(); err != nil {
			logger.Error("卸载服务失败:", err)
			os.Exit(1)
		}
		logger.Info("服务卸载成功")
		return
	}

	// 运行 Pilot
	if err := run(cfg); err != nil {
		logger.Error("Pilot 运行失败:", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("正在启动 UBAX-Pilot...")
	logger.Info("平台:", cfg.OS)
	logger.Info("Vector 路径:", cfg.VectorBinPath)

	// 1. 初始化配置渲染器
	renderer := comm.NewConfigRenderer(cfg.VectorConfPath)

	// 2. 初始化远程配置客户端（拉取 → 渲染 → 热重载）
	fetcher := comm.NewHTTPConfigFetcher(cfg.ServerEndpoint)
	remoteConfig := comm.NewRemoteConfigClient(
		fetcher,
		renderer,
		time.Duration(cfg.HeartbeatInterval)*time.Second,
	)

	// 3. 初始化进程管理器（Vector 进程生命周期管理）
	processManager := control.NewProcessManager(cfg)

	// 4. 初始化服务端推送客户端（配置推送 + 远程命令）
	pushClient := comm.NewServerPushClient(cfg.ServerEndpoint, "")
	pushClient.SetConfigHandler(func(rules []byte) error {
		if err := renderer.Render(rules); err != nil {
			return err
		}
		return renderer.TriggerHotReload()
	})
	pushClient.SetCommandHandler(func(cmd comm.CommandPayload) error {
		return handleRemoteCommand(cmd, processManager, renderer)
	})

	// 5. 初始化心跳上报（携带采集器实时状态）
	heartbeat := comm.NewHeartbeatReporter(cfg, processManager)

	// 6. 初始化资源监控（内存/CPU 超限自动重启）
	resourceMonitor := control.NewResourceMonitor(cfg, processManager)
	resourceMonitor.Start()
	defer resourceMonitor.Stop()

	// 启动 Vector 进程
	if err := processManager.Start(ctx); err != nil {
		return err
	}
	defer func(processManager *control.ProcessManager) {
		_ = processManager.Stop()
	}(processManager)

	// 启动心跳上报
	heartbeat.Start(ctx)
	defer heartbeat.Stop()

	// 启动远程配置轮询
	remoteConfig.StartPolling(ctx)
	defer remoteConfig.Stop()

	// 启动服务端推送连接
	pushClient.Start(ctx)
	defer pushClient.Stop()

	// 等待关闭信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("UBAX-Pilot 正在运行，按 Ctrl+C 停止。")
	<-sigCh

	logger.Info("正在关闭 UBAX-Pilot...")
	cancel()

	return nil
}

// handleRemoteCommand 处理服务端下发的远程命令
func handleRemoteCommand(cmd comm.CommandPayload, pm *control.ProcessManager, renderer *comm.ConfigRenderer) error {
	switch cmd.Action {
	case "restart":
		logger.Info("收到重启命令")
		return pm.Restart()

	case "stop":
		logger.Info("收到停止命令")
		return pm.Stop()

	case "reload":
		logger.Info("收到配置重载命令")
		return pm.ReloadConfig()

	case "upgrade":
		logger.Info("收到升级命令，版本:", cmd.Params["version"])
		// TODO: 实现二进制下载和替换逻辑
		logger.Warn("升级功能尚未实现")
		return nil

	default:
		return fmt.Errorf("未知命令: %s", cmd.Action)
	}
}
