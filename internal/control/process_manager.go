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
	cfg        *config.Config
	ctx        context.Context
	cancel     context.CancelFunc
	cmd        *exec.Cmd
	mu         sync.Mutex
	running    bool
	stopCh     chan struct{}
	doneCh     chan struct{}
	restartCh  chan struct{}
}

// NewProcessManager 创建一个新的进程管理器
func NewProcessManager(cfg *config.Config) *ProcessManager {
	return &ProcessManager{
		cfg:       cfg,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
		restartCh: make(chan struct{}, 1),
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
	if !pm.running {
		pm.mu.Unlock()
		return nil
	}

	// 取消上下文，终止子进程
	pm.cancel()

	close(pm.stopCh)

	pm.running = false
	pm.mu.Unlock()

	<-pm.doneCh // 等待监控协程退出

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

	// 等待旧监控协程退出
	<-pm.doneCh

	// 创建新上下文并启动
	pm.ctx, pm.cancel = context.WithCancel(context.Background())
	if err := pm.startProcess(); err != nil {
		return err
	}

	// 重置通道
	pm.doneCh = make(chan struct{})
	pm.stopCh = make(chan struct{})

	// 启动新监控协程
	go pm.monitor()

	logger.Info("Vector 已重启，PID:", pm.cmd.Process.Pid)
	return nil
}

// monitor 监控 Vector 进程，退出时不自动重启
func (pm *ProcessManager) monitor() {
	defer close(pm.doneCh)

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

		// 如果是上下文取消（正常关闭），直接退出
		if pm.ctx.Err() != nil {
			pm.running = false
			pm.mu.Unlock()
			logger.Info("Vector 已正常退出")
			return
		}

		pm.running = false
		pm.mu.Unlock()

		logger.Error("Vector 异常退出:", err)
		return
	}
}

// ResourceMonitor 跟踪 CPU 和内存使用情况（30s间隔）
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
		checkInterval: 30 * time.Second,
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
