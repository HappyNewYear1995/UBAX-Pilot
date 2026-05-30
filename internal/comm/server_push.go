package comm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	Action string                 `json:"action"` // "restart" | "stop"
	Params map[string]interface{} `json:"params"`
}

// ConfigPayload 配置推送载荷
type ConfigPayload struct {
	Rules   json.RawMessage `json:"rules"`
	Version string          `json:"version"`
}

// ServerPushClient 接收服务端主动推送的配置和命令
type ServerPushClient struct {
	endpoint        string
	agentUUID       string
	token           string
	configHandler   func([]byte) error
	commandHandler  func(CommandPayload) error
	onConnected     func()
	conn            *http.Response
	stopCh          chan struct{}
	mu              sync.Mutex
	connected       bool
	reconnectDelay  time.Duration
	maxDelay        time.Duration
}

// NewServerPushClient 创建服务端推送客户端
func NewServerPushClient(endpoint, agentUUID, token string) *ServerPushClient {
	return &ServerPushClient{
		endpoint:       endpoint,
		agentUUID:      agentUUID,
		token:          token,
		stopCh:         make(chan struct{}),
		reconnectDelay: 10 * time.Second,
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

// SetOnConnected 设置连接建立后的回调
func (sp *ServerPushClient) SetOnConnected(handler func()) {
	sp.onConnected = handler
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
			logger.Errorf("推送连接断开: %v，%v 后重连", err, delay)
			// 指数退避
			delay *= 2
			if delay > sp.maxDelay {
				delay = sp.maxDelay
			}
		} else {
			// 连接正常关闭（非主动停止），立即重连
			select {
			case <-sp.stopCh:
				return
			default:
				logger.Info("推送连接已关闭，正在重连...")
				delay = sp.reconnectDelay
			}
		}
	}
}

func (sp *ServerPushClient) connect(ctx context.Context) error {
	logger.Info("正在连接推送服务:", sp.endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", sp.endpoint+"/pilot/agent/content", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sp.token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-Agent-UUID", sp.agentUUID)

	client := &http.Client{Timeout: 0} // 长连接不设置超时
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	sp.mu.Lock()
	sp.connected = true
	sp.mu.Unlock()

	logger.Info("推送连接已建立")

	if sp.onConnected != nil {
		sp.onConnected()
	}

	// 读取 SSE 推送消息
	buf := make([]byte, 4096)
	var eventBuf string
	var eventType string

	for {
		select {
		case <-sp.stopCh:
			return nil
		case <-ctx.Done():
			return nil
		default:
		}

		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			sp.mu.Lock()
			sp.connected = false
			sp.mu.Unlock()
			return fmt.Errorf("读取消息失败: %w", err)
		}
		if n == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		eventBuf += string(buf[:n])

		// 按行解析 SSE 事件
		for {
			idx := indexOfAny(eventBuf, []string{"\r\n\r\n", "\n\n"})
			if idx == -1 {
				break
			}

			block := eventBuf[:idx]
			eventBuf = eventBuf[idx+2:]

			var msg PushMessage
			lines := splitLines(block)
			for _, line := range lines {
				if strings.HasPrefix(line, "event:") {
					eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				} else if strings.HasPrefix(line, "data:") {
					data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
					if eventType == "message" {
						if err := json.Unmarshal([]byte(data), &msg); err != nil {
							logger.Error("解析推送消息失败:", err, "原始数据:", data)
							continue
						}
						sp.handleMessage(msg)
					} else if eventType == "connected" {
						logger.Info("服务端确认连接:", data)
					}
				}
			}
			eventType = ""
		}

		if err == io.EOF {
			sp.mu.Lock()
			sp.connected = false
			sp.mu.Unlock()
			return fmt.Errorf("连接已关闭")
		}
	}
}

func indexOfAny(s string, subs []string) int {
	minIdx := -1
	for _, sub := range subs {
		if idx := strings.Index(s, sub); idx != -1 {
			if minIdx == -1 || idx < minIdx {
				minIdx = idx
			}
		}
	}
	return minIdx
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
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

	logger.Info("收到配置推送，版本:", cfg.Version, "，Vector 将自动重载")

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
