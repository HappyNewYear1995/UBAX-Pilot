package comm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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
	cfg            *config.Config
	statusProvider CollectorStatusProvider
	interval       time.Duration
	stopCh         chan struct{}
	httpClient     *http.Client
	heartbeatCh    chan struct{}
}

// HeartbeatPayload 心跳上报数据
type HeartbeatPayload struct {
	AgentUUID       string    `json:"agent_uuid"`
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
		heartbeatCh:    make(chan struct{}, 1),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start 开始周期性心跳上报
func (hr *HeartbeatReporter) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(hr.interval)
		defer ticker.Stop()

		for {
			select {
			case <-hr.heartbeatCh:
				hr.sendHeartbeat(ctx)
				ticker.Reset(hr.interval)
			case <-ticker.C:
				hr.sendHeartbeat(ctx)
			case <-hr.stopCh:
				return
			}
		}
	}()
}

// Trigger 触发一次心跳（通常在SSE连接建立后调用）
func (hr *HeartbeatReporter) Trigger() {
	select {
	case hr.heartbeatCh <- struct{}{}:
	default:
	}
}

// Stop 停止心跳上报
func (hr *HeartbeatReporter) Stop() {
	close(hr.stopCh)
}

// 发送心跳
func (hr *HeartbeatReporter) sendHeartbeat(ctx context.Context) {
	collectorRunning := false
	if hr.statusProvider != nil {
		collectorRunning = hr.statusProvider.IsRunning()
	}

	hostname, _ := os.Hostname()

	payload := &HeartbeatPayload{
		AgentUUID:       hr.cfg.AgentUUID,
		Version:         config.GetVersion(),
		Hostname:        hostname,
		Timestamp:       time.Now(),
		Healthy:         collectorRunning,
		CollectorStatus: mapStatus(collectorRunning),
		OS:              hr.cfg.OS,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error("序列化心跳失败:", err)
		return
	}

	url := hr.cfg.ServerEndpoint + "/pilot/agent/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("创建心跳请求失败:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hostname", hostname)

	resp, err := hr.httpClient.Do(req)
	if err != nil {
		logger.Error("发送心跳失败:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("心跳请求被拒绝: %s", resp.Status)
		return
	}

	logger.Debugf("心跳已发送: UUID = %s, 版本=%s, 采集器=%s", payload.AgentUUID, payload.Version, payload.CollectorStatus)
}

func mapStatus(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}
