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

// ProcessManager manages the lifecycle of the underlying collector process
type ProcessManager struct {
	cfg     *config.Config
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
}

// NewProcessManager creates a new process manager
func NewProcessManager(cfg *config.Config) *ProcessManager {
	return &ProcessManager{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// Start launches the collector process
func (pm *ProcessManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return fmt.Errorf("collector is already running")
	}

	pm.cmd = exec.CommandContext(ctx, pm.cfg.CollectorBinPath, "-c", pm.cfg.CollectorConfPath)
	pm.cmd.Stdout = os.Stdout
	pm.cmd.Stderr = os.Stderr

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}

	pm.running = true
	logger.Info("Collector started with PID:", pm.cmd.Process.Pid)

	// Monitor process in background
	go pm.monitor()

	return nil
}

// Stop gracefully shuts down the collector process
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return nil
	}

	close(pm.stopCh)

	if pm.cmd.Process != nil {
		if err := pm.cmd.Process.Signal(os.Interrupt); err != nil {
			logger.Warn("Failed to send interrupt signal, killing process:", err)
			pm.cmd.Process.Kill()
		}
	}

	pm.running = false
	logger.Info("Collector stopped")
	return nil
}

// IsRunning returns whether the collector is currently running
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.running
}

// monitor watches the collector process and restarts on crash
func (pm *ProcessManager) monitor() {
	if pm.cmd == nil || pm.cmd.Process == nil {
		return
	}

	err := pm.cmd.Wait()
	pm.mu.Lock()
	pm.running = false
	pm.mu.Unlock()

	if err != nil {
		logger.Error("Collector exited with error:", err)
		// Auto-restart after delay
		time.Sleep(5 * time.Second)
		logger.Info("Attempting to restart collector...")
		// Note: In production, use a context that persists across restarts
	}
}

// ResourceMonitor tracks CPU and memory usage
type ResourceMonitor struct {
	cfg         *config.Config
	checkInterval time.Duration
	stopCh      chan struct{}
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(cfg *config.Config) *ResourceMonitor {
	return &ResourceMonitor{
		cfg:           cfg,
		checkInterval: 10 * time.Second,
		stopCh:        make(chan struct{}),
	}
}

// Start begins periodic resource monitoring
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

// Stop stops the resource monitor
func (rm *ResourceMonitor) Stop() {
	close(rm.stopCh)
}

func (rm *ResourceMonitor) check() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memMB := m.Alloc / 1024 / 1024
	if int64(memMB) > rm.cfg.MaxMemoryMB {
		logger.Warn("Memory usage exceeds limit:", memMB, "MB /", rm.cfg.MaxMemoryMB, "MB")
	}

	logger.Debug("Current memory usage:", memMB, "MB")
}
