package comm

import (
	"context"
	"os"
	"time"

	"github.com/ubax/ubax-pilot/pkg/config"
	"github.com/ubax/ubax-pilot/pkg/logger"
)

// CollectorStatusProvider 提供采集器状态的接口
type CollectorStatusProvider interface {
	IsRunning() bool
}

// HeartbeatReporter 定期向服务端发送健康状态
type HeartbeatReporter struct {
	cfg              *config.Config
	statusProvider   CollectorStatusProvider
	interval         time.Duration
	stopCh           chan struct{}
}

// HeartbeatPayload 心跳上报数据
type HeartbeatPayload struct {
	Version         string    `json:"version"`
	Hostname        string    `json:"hostname"`
	Timestamp       time.Time `json:"timestamp"`
	Healthy         bool      `json:"healthy"`
	CollectorStatus string    `json:"collector_status"`
	OS              string    `json:"os"`
}

// NewHeartbeatReporter 创建一个新的心跳上报器
func NewHeartbeatReporter(cfg *config.Config, statusProvider CollectorStatusProvider) *HeartbeatReporter {
	return &HeartbeatReporter{
		cfg:            cfg,
		statusProvider: statusProvider,
		interval:       time.Duration(cfg.HeartbeatInterval) * time.Second,
		stopCh:         make(chan struct{}),
	}
}

// Start 开始周期性心跳上报
func (hr *HeartbeatReporter) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(hr.interval)
		defer ticker.Stop()

		// 发送初始心跳
		hr.sendHeartbeat(ctx)

		for {
			select {
			case <-ticker.C:
				hr.sendHeartbeat(ctx)
			case <-hr.stopCh:
				return
			}
		}
	}()
}

// Stop 停止心跳上报
func (hr *HeartbeatReporter) Stop() {
	close(hr.stopCh)
}

func (hr *HeartbeatReporter) sendHeartbeat(ctx context.Context) {
	collectorRunning := false
	if hr.statusProvider != nil {
		collectorRunning = hr.statusProvider.IsRunning()
	}

	hostname, _ := os.Hostname()

	payload := &HeartbeatPayload{
		Version:         config.GetVersion(),
		Hostname:        hostname,
		Timestamp:       time.Now(),
		Healthy:         collectorRunning,
		CollectorStatus: mapStatus(collectorRunning),
		OS:              hr.cfg.OS,
	}

	// TODO: 通过 HTTP/gRPC 将数据发送到服务端
	logger.Debug("心跳已发送:", payload.Version, "采集器状态:", payload.CollectorStatus)
}

func mapStatus(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}
