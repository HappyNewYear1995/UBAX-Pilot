package comm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ubax/ubax-pilot/pkg/logger"
)

// PushMessage 服务端推送消息
type PushMessage struct {
	Type      string          `json:"type"` // "config" | "command"
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// CommandPayload 远程命令载荷
type CommandPayload struct {
	Action string                 `json:"action"` // "restart" | "stop" | "upgrade" | "reload"
	Params map[string]interface{} `json:"params"`
}

// ConfigPayload 配置推送载荷
type ConfigPayload struct {
	Rules   json.RawMessage `json:"rules"`
	Version string          `json:"version"`
}

// ServerPushClient 接收服务端主动推送的配置和命令
type ServerPushClient struct {
	endpoint       string
	token          string
	configHandler  func([]byte) error
	commandHandler func(CommandPayload) error
	conn           *http.Response
	stopCh         chan struct{}
	mu             sync.Mutex
	connected      bool
	reconnectDelay time.Duration
	maxDelay       time.Duration
}

// NewServerPushClient 创建服务端推送客户端
func NewServerPushClient(endpoint, token string) *ServerPushClient {
	return &ServerPushClient{
		endpoint:       endpoint,
		token:          token,
		stopCh:         make(chan struct{}),
		reconnectDelay: 3 * time.Second,
		maxDelay:       60 * time.Second,
	}
}

// SetConfigHandler 设置配置推送处理器
func (sp *ServerPushClient) SetConfigHandler(handler func([]byte) error) {
	sp.configHandler = handler
}

// SetCommandHandler 设置命令推送处理器
func (sp *ServerPushClient) SetCommandHandler(handler func(CommandPayload) error) {
	sp.commandHandler = handler
}

// Start 启动推送连接
func (sp *ServerPushClient) Start(ctx context.Context) {
	go sp.connectLoop(ctx)
}

// Stop 停止推送连接
func (sp *ServerPushClient) Stop() {
	close(sp.stopCh)
}

func (sp *ServerPushClient) connectLoop(ctx context.Context) {
	delay := sp.reconnectDelay

	for {
		select {
		case <-sp.stopCh:
			return
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		if err := sp.connect(ctx); err != nil {
			logger.Error("推送连接失败:", err)
			// 指数退避
			delay *= 2
			if delay > sp.maxDelay {
				delay = sp.maxDelay
			}
		} else {
			delay = sp.reconnectDelay
		}
	}
}

func (sp *ServerPushClient) connect(ctx context.Context) error {
	// 当前使用 HTTP 长轮询模拟
	logger.Info("正在连接推送服务:", sp.endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", sp.endpoint+"/pilot/agent/push", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sp.token)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0} // 长连接不设置超时
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	sp.mu.Lock()
	sp.connected = true
	sp.mu.Unlock()

	logger.Info("推送连接已建立")

	// 读取推送消息（简化实现，实际应使用 WebSocket 或 SSE 解析器）
	decoder := json.NewDecoder(resp.Body)
	for {
		select {
		case <-sp.stopCh:
			return nil
		case <-ctx.Done():
			return nil
		default:
		}

		var msg PushMessage
		if err := decoder.Decode(&msg); err != nil {
			sp.mu.Lock()
			sp.connected = false
			sp.mu.Unlock()
			return fmt.Errorf("读取消息失败: %w", err)
		}

		sp.handleMessage(msg)
	}
}

func (sp *ServerPushClient) handleMessage(msg PushMessage) {
	switch msg.Type {
	case "config":
		sp.handleConfigPush(msg.Payload)
	case "command":
		sp.handleCommand(msg.Payload)
	default:
		logger.Warn("未知推送消息类型:", msg.Type)
	}
}

func (sp *ServerPushClient) handleConfigPush(payload json.RawMessage) {
	var cfg ConfigPayload
	if err := json.Unmarshal(payload, &cfg); err != nil {
		logger.Error("解析配置推送失败:", err)
		return
	}

	logger.Info("收到配置推送，版本:", cfg.Version)

	if sp.configHandler != nil {
		if err := sp.configHandler(cfg.Rules); err != nil {
			logger.Error("处理配置推送失败:", err)
		}
	}
}

func (sp *ServerPushClient) handleCommand(payload json.RawMessage) {
	var cmd CommandPayload
	if err := json.Unmarshal(payload, &cmd); err != nil {
		logger.Error("解析命令推送失败:", err)
		return
	}

	logger.Info("收到远程命令:", cmd.Action)

	if sp.commandHandler != nil {
		if err := sp.commandHandler(cmd); err != nil {
			logger.Error("执行远程命令失败:", err)
		}
	}
}

// IsConnected 返回推送连接状态
func (sp *ServerPushClient) IsConnected() bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.connected
}
