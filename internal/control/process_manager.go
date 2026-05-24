package control

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/ubax/ubax-pilot/pkg/config"
	"github.com/ubax/ubax-pilot/pkg/logger"
)

// ProcessManager 管理底层 Vector 进程的生命周期
type ProcessManager struct {
	cfg         *config.Config
	ctx         context.Context
	cancel      context.CancelFunc
	cmd         *exec.Cmd
	mu          sync.Mutex
	running     bool
	stopCh      chan struct{}
	restartDone chan struct{}
}

// NewProcessManager 创建一个新的进程管理器
func NewProcessManager(cfg *config.Config) *ProcessManager {
	return &ProcessManager{
		cfg:         cfg,
		stopCh:      make(chan struct{}),
		restartDone: make(chan struct{}),
	}
}

// Start 启动 Vector 进程
func (pm *ProcessManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return fmt.Errorf("Vector 已在运行")
	}

	// 创建可取消的子上下文用于进程管理
	pm.ctx, pm.cancel = context.WithCancel(ctx)

	if err := pm.startProcess(); err != nil {
		return err
	}

	pm.running = true

	// 后台监控进程
	go pm.monitor()

	return nil
}

// startProcess 启动 Vector 子进程
func (pm *ProcessManager) startProcess() error {
	pm.cmd = exec.CommandContext(pm.ctx, pm.cfg.VectorBinPath, "--config", pm.cfg.VectorConfPath, "--quiet", "--watch-config")
	pm.cmd.Stdout = os.Stdout
	pm.cmd.Stderr = os.Stderr

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("启动 Vector 失败: %w", err)
	}

	logger.Info("Vector 已启动，PID:", pm.cmd.Process.Pid)
	return nil
}

// Stop 优雅关闭 Vector 进程
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return nil
	}

	// 取消上下文，终止子进程
	pm.cancel()

	close(pm.stopCh)

	pm.running = false
	<-pm.restartDone // 等待监控协程退出

	logger.Info("Vector 采集器已停止")
	return nil
}

// IsRunning 返回采集器是否正在运行
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.running
}

// Restart 重启 Vector 进程
func (pm *ProcessManager) Restart() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return pm.startProcess()
	}

	// 取消旧上下文
	pm.cancel()
	<-pm.restartDone // 等待旧监控协程退出

	// 创建新上下文并启动
	pm.ctx, pm.cancel = context.WithCancel(pm.ctx)
	if err := pm.startProcess(); err != nil {
		return err
	}

	// 启动新监控协程
	go pm.monitor()

	logger.Info("Vector 已重启，PID:", pm.cmd.Process.Pid)
	return nil
}

// ReloadConfig 触发 Vector 配置热重载
func (pm *ProcessManager) ReloadConfig() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running || pm.cmd.Process == nil {
		return fmt.Errorf("Vector 未运行，无法重载配置")
	}

	// Vector 支持通过 SIGHUP 信号触发热重载
	if err := pm.cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("发送重载信号失败: %w", err)
	}

	logger.Info("已发送 Vector 配置重载信号")
	return nil
}

// monitor 监控 Vector 进程，崩溃时自动重启
func (pm *ProcessManager) monitor() {
	defer close(pm.restartDone)

	for {
		if pm.cmd == nil || pm.cmd.Process == nil {
			return
		}

		err := pm.cmd.Wait()

		pm.mu.Lock()
		if !pm.running {
			pm.mu.Unlock()
			return
		}

		// 如果是上下文取消（正常关闭），不重启
		if pm.ctx.Err() != nil {
			pm.running = false
			pm.mu.Unlock()
			logger.Info("Vector 已正常退出")
			return
		}

		pm.running = false
		pm.mu.Unlock()

		logger.Error("Vector 异常退出:", err)

		// 指数退避重启
		retryDelay := 5 * time.Second
		maxDelay := 60 * time.Second
		for attempt := 1; ; attempt++ {
			select {
			case <-pm.stopCh:
				logger.Info("收到停止信号，取消重启")
				return
			case <-time.After(retryDelay):
			}

			logger.Info("正在尝试重启 Vector (第 %d 次)...", attempt)

			pm.mu.Lock()
			if err := pm.startProcess(); err != nil {
				pm.mu.Unlock()
				logger.Error("重启 Vector 失败:", err)
				// 指数退避，最大 60 秒
				retryDelay *= 2
				if retryDelay > maxDelay {
					retryDelay = maxDelay
				}
				continue
			}

			pm.running = true
			pm.mu.Unlock()

			logger.Info("Vector 重启成功，PID:", pm.cmd.Process.Pid)
			// 继续监控新进程
			break
		}
	}
}

// ResourceMonitor 跟踪 CPU 和内存使用情况
type ResourceMonitor struct {
	cfg           *config.Config
	pm            *ProcessManager
	checkInterval time.Duration
	stopCh        chan struct{}
}

// NewResourceMonitor 创建一个新的资源监控器
func NewResourceMonitor(cfg *config.Config, pm *ProcessManager) *ResourceMonitor {
	return &ResourceMonitor{
		cfg:           cfg,
		pm:            pm,
		checkInterval: 10 * time.Second,
		stopCh:        make(chan struct{}),
	}
}

// Start 开始周期性资源监控
func (rm *ResourceMonitor) Start() {
	go func() {
		ticker := time.NewTicker(rm.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rm.check()
			case <-rm.stopCh:
				return
			}
		}
	}()
}

// Stop 停止资源监控
func (rm *ResourceMonitor) Stop() {
	close(rm.stopCh)
}

func (rm *ResourceMonitor) check() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memMB := m.Alloc / 1024 / 1024
	if int64(memMB) > rm.cfg.MaxMemoryMB {
		logger.Warn("内存使用超出限制:", memMB, "MB /", rm.cfg.MaxMemoryMB, "MB")
	}

	logger.Debug("当前内存使用:", memMB, "MB")
}
