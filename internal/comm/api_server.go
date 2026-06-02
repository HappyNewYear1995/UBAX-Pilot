package comm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ubax/ubax-pilot/pkg/config"
	"github.com/ubax/ubax-pilot/pkg/logger"
)

type CommandPayload struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type ConfigPayload struct {
	Rules   json.RawMessage `json:"rules"`
	Version int64           `json:"version"`
}

type RegisterPayload struct {
	Hostname     string `json:"hostname"`
	IP           string `json:"ip"`
	OS           string `json:"os"`
	AgentVersion string `json:"version"`
	UUID         string `json:"uuid"`
	TerminalType string `json:"terminal"`
}

type APIServer struct {
	cfg            *config.Config
	configHandler  func([]byte) error
	commandHandler func(CommandPayload) error
	server         *http.Server
	stopCh         chan struct{}
	mu             sync.Mutex
	running        bool
}

func NewAPIServer(cfg *config.Config) *APIServer {
	return &APIServer{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

func (s *APIServer) Register() error {
	payload := &RegisterPayload{
		IP:           getLocalIP(),
		Hostname:     getHostname(),
		OS:           getOSDetail(),
		AgentVersion: config.GetVersion(),
		UUID:         s.cfg.AgentUUID,
		TerminalType: getTerminalType(s.cfg),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化注册请求体失败: %w", err)
	}

	reqURL := s.cfg.ServerEndpoint + "/gather/agent/register"
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建注册请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-UUID", s.cfg.AgentUUID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("注册请求失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("注册被拒绝: %s", resp.Status)
	}

	logger.Info("Agent 注册成功")
	return nil
}

func (s *APIServer) SetConfigHandler(handler func([]byte) error) {
	s.configHandler = handler
}

func (s *APIServer) SetCommandHandler(handler func(CommandPayload) error) {
	s.commandHandler = handler
}

func (s *APIServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/config", s.handleConfigUpdate)
	mux.HandleFunc("/api/command", s.handleCommandExecute)
	mux.HandleFunc("/api/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    ":19090",
		Handler: mux,
	}

	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("监听端口失败: %w", err)
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	logger.Info("API 服务器已启动:", s.server.Addr)

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("API 服务器异常:", err)
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			s.Stop()
		case <-s.stopCh:
		}
	}()

	return nil
}

func (s *APIServer) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		logger.Error("API 服务器关闭失败:", err)
	} else {
		logger.Info("API 服务器已关闭")
	}
}

func (s *APIServer) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload ConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Error("解析配置请求失败:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	logger.Info("收到配置更新请求")

	if s.configHandler != nil {
		if err := s.configHandler(payload.Rules); err != nil {
			logger.Error("处理配置更新失败:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) handleCommandExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd CommandPayload
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		logger.Error("解析命令请求失败:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	logger.Info("收到远程命令:", cmd.Action)

	if s.commandHandler != nil {
		if err := s.commandHandler(cmd); err != nil {
			logger.Error("执行远程命令失败:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "unknown"
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getOSDetail() string {
	if runtime.GOOS == "windows" {
		return getWindowsVersion()
	}
	return getUnixVersion()
}

func getWindowsVersion() string {
	cmd := exec.Command("powershell", "-Command",
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; "+
			"(Get-CimInstance Win32_OperatingSystem).Caption + ' ' + (Get-CimInstance Win32_OperatingSystem).Version")
	out, err := cmd.Output()
	if err != nil {
		return "Windows " + runtime.GOARCH
	}
	return strings.TrimSpace(string(out))
}

func getUnixVersion() string {
	cmd := exec.Command("uname", "-sr")
	out, err := cmd.Output()
	if err != nil {
		return runtime.GOOS + " " + runtime.GOARCH
	}
	return strings.TrimSpace(string(out))
}

func getTerminalType(cfg *config.Config) string {
	if cfg.IsWindows() {
		return "Windows"
	}
	return "Linux"
}
