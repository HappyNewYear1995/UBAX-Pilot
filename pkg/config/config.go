package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config 存储 UBAX-Pilot 的全局配置
type Config struct {
	// Agent 唯一标识
	AgentUUID string `yaml:"agent_uuid"`

	// 服务端连接设置
	ServerEndpoint    string `yaml:"server_endpoint"`
	HeartbeatInterval int    `yaml:"heartbeat_interval_seconds"`

	// Vector 采集器设置
	VectorBinPath  string `yaml:"vector_bin_path"`
	VectorConfPath string `yaml:"vector_conf_path"`

	// 资源限制
	MaxMemoryMB int64 `yaml:"max_memory_mb"`

	// 日志设置
	LogLevel string `yaml:"log_level"`
	LogFile  string `yaml:"log_file"`

	// 平台相关
	OS string `yaml:"-"`
}

// DefaultConfig 返回带有合理默认值的配置
func DefaultConfig() *Config {
	return &Config{
		AgentUUID:         generateUUID(),
		ServerEndpoint:    "http://localhost:48080/admin-api",
		HeartbeatInterval: 60,
		VectorBinPath:     defaultVectorBinPath(),
		VectorConfPath:    defaultVectorConfPath(),
		MaxMemoryMB:       512,
		LogLevel:          "info",
		LogFile:           defaultLogPath(),
		OS:                runtime.GOOS,
	}
}

// generateUUID 生成 UUID v4
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func defaultVectorBinPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Program Files\Vector\bin\vector.exe`
	}
	return "/usr/local/bin/vector"
}

func defaultVectorConfPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Program Files\Vector\config\vector.yaml`
	}
	return "/etc/ubax-pilot/vector.yaml"
}

// LoadConfig 从文件加载配置，回退到默认值
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	// 如果未指定路径，使用默认路径
	if path == "" {
		path = defaultConfigPath()
	}

	// 如果配置文件存在，则加载
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		// 配置文件不存在，生成默认配置文件
		if err := generateDefaultConfig(path); err != nil {
			return nil, err
		}
	}

	// 确保 agent_uuid 已持久化（首次运行生成后不再修改）
	if cfg.AgentUUID == "" {
		cfg.AgentUUID = generateUUID()
		_ = saveConfig(path, cfg)
	}

	// 确保平台相关字段正确设置
	cfg.OS = runtime.GOOS

	// 如果某些字段仍为空，使用默认值填充
	if cfg.VectorBinPath == "" {
		cfg.VectorBinPath = defaultVectorBinPath()
	}
	if cfg.VectorConfPath == "" {
		cfg.VectorConfPath = defaultVectorConfPath()
	}

	return cfg, nil
}

// generateDefaultConfig 生成默认配置文件
func generateDefaultConfig(path string) error {
	cfg := DefaultConfig()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// saveConfig 将配置保存到文件
func saveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// defaultConfigPath 返回默认配置文件路径
func defaultConfigPath() string {
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "ubax-pilot", "config", "config.yaml")
	}
	return "/etc/ubax-pilot/config.yaml"
}

func defaultLogPath() string {
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "ubax-pilot", "logs", "ubax-pilot.log")
	}
	return "/var/log/ubax-pilot/ubax-pilot.log"
}

// IsWindows 判断是否运行在 Windows 上
func (c *Config) IsWindows() bool {
	return c.OS == "windows"
}

// IsLinux 判断是否运行在 Linux 上
func (c *Config) IsLinux() bool {
	return c.OS == "linux"
}
