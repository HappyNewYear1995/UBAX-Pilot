package comm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ubax/ubax-pilot/pkg/logger"
)

// ConfigFetcher 定义获取远程配置的接口
type ConfigFetcher interface {
	Fetch(ctx context.Context) ([]byte, error)
}

// HTTPConfigFetcher 通过 HTTP 从服务端获取配置
type HTTPConfigFetcher struct {
	endpoint   string
	httpClient *http.Client
}

// NewHTTPConfigFetcher 创建 HTTP 配置获取器
func NewHTTPConfigFetcher(endpoint string) *HTTPConfigFetcher {
	return &HTTPConfigFetcher{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Fetch 从服务端拉取配置
func (hf *HTTPConfigFetcher) Fetch(ctx context.Context) ([]byte, error) {
	url := hf.endpoint + "/api/config"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	hostname, _ := os.Hostname()
	req.Header.Set("X-Hostname", hostname)

	resp, err := hf.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求配置失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务端返回错误: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	logger.Infof("成功从服务端拉取配置: %d 字节", len(body))
	return body, nil
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
