package comm

import (
	"context"
	"fmt"
	"time"

	"github.com/ubax/ubax-pilot/pkg/logger"
)

// ConfigFetcher 定义获取远程配置的接口
type ConfigFetcher interface {
	Fetch(ctx context.Context) ([]byte, error)
}

// HTTPConfigFetcher 通过 HTTP 从服务端获取配置
type HTTPConfigFetcher struct {
	endpoint string
}

// NewHTTPConfigFetcher 创建 HTTP 配置获取器
func NewHTTPConfigFetcher(endpoint string) *HTTPConfigFetcher {
	return &HTTPConfigFetcher{endpoint: endpoint}
}

// Fetch 从服务端拉取配置
func (hf *HTTPConfigFetcher) Fetch(ctx context.Context) ([]byte, error) {
	// TODO: 实现实际的 HTTP 请求
	logger.Info("正在从以下地址获取配置:", hf.endpoint)
	return nil, fmt.Errorf("未实现")
}

// RemoteConfigClient 处理与 UBAX 服务端的通信
type RemoteConfigClient struct {
	fetcher      ConfigFetcher
	renderer     *ConfigRenderer
	pollInterval time.Duration
	stopCh       chan struct{}
}

// NewRemoteConfigClient 创建一个新的远程配置客户端
func NewRemoteConfigClient(fetcher ConfigFetcher, renderer *ConfigRenderer, pollInterval time.Duration) *RemoteConfigClient {
	return &RemoteConfigClient{
		fetcher:      fetcher,
		renderer:     renderer,
		pollInterval: pollInterval,
		stopCh:       make(chan struct{}),
	}
}

// StartPolling 开始周期性配置轮询
func (rc *RemoteConfigClient) StartPolling(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(rc.pollInterval)
		defer ticker.Stop()

		// 启动时立即拉取一次
		rc.pullAndRender(ctx)

		for {
			select {
			case <-ticker.C:
				rc.pullAndRender(ctx)
			case <-rc.stopCh:
				return
			}
		}
	}()
}

// Stop 停止配置轮询
func (rc *RemoteConfigClient) Stop() {
	close(rc.stopCh)
}

func (rc *RemoteConfigClient) pullAndRender(ctx context.Context) {
	data, err := rc.fetcher.Fetch(ctx)
	if err != nil {
		logger.Error("获取远程配置失败:", err)
		return
	}

	if err := rc.renderer.Render(data); err != nil {
		logger.Error("渲染配置失败:", err)
		return
	}

	if err := rc.renderer.TriggerHotReload(); err != nil {
		logger.Error("触发热重载失败:", err)
	}
}
