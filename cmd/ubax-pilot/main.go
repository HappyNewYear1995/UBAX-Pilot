package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	if err := logger.InitLogger(cfg.LogLevel, cfg.LogFile); err != nil {
		return err
	}
	defer logger.Close()

	logger.Info("正在启动 UBAX-Pilot...")
	logger.Info("平台:", cfg.OS)
	logger.Info("Vector 路径:", cfg.VectorBinPath)

	// 1. 初始化配置渲染器
	renderer := comm.NewConfigRenderer(cfg.VectorConfPath)

	// 2. 初始化进程管理器（Vector 进程生命周期管理）
	processManager := control.NewProcessManager(cfg)

	// 3. 初始化 API 服务器（接收服务端配置推送 + 远程命令）
	apiServer := comm.NewAPIServer(cfg)
	apiServer.SetConfigHandler(func(rules []byte) error {
		return renderer.Render(rules)
	})
	apiServer.SetCommandHandler(func(cmd comm.CommandPayload) error {
		return handleRemoteCommand(cmd, processManager)
	})

	// 向服务端注册
	if err := apiServer.Register(); err != nil {
		logger.Warn("Agent 注册失败:", err)
	}

	// 4. 初始化心跳上报（携带采集器实时状态）
	heartbeat := comm.NewHeartbeatReporter(cfg, processManager)

	// 5. 初始化资源监控（内存/CPU 超限自动重启）
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

	// 启动 API 服务器
	if err := apiServer.Start(ctx); err != nil {
		return err
	}
	defer apiServer.Stop()

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
func handleRemoteCommand(cmd comm.CommandPayload, pm *control.ProcessManager) error {
	switch cmd.Action {
	case "restart":
		logger.Info("收到重启命令")
		return pm.Restart()

	case "stop":
		logger.Info("收到停止命令")
		return pm.Stop()

	default:
		return fmt.Errorf("未知命令: %s", cmd.Action)
	}
}
