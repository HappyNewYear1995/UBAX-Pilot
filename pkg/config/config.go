package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config 存储 UBAX-Pilot 的全局配置
type Config struct {
	// 服务端连接设置
	ServerEndpoint    string `yaml:"server_endpoint"`
	HeartbeatInterval int    `yaml:"heartbeat_interval_seconds"`

	// Vector 采集器设置
	VectorBinPath  string `yaml:"vector_bin_path"`
	VectorConfPath string `yaml:"vector_conf_path"`

	// 资源限制
	MaxMemoryMB int64 `yaml:"max_memory_mb"`

	// 平台相关
	OS string `yaml:"-"`
}

// DefaultConfig 返回带有合理默认值的配置
func DefaultConfig() *Config {
	return &Config{
		ServerEndpoint:    "localhost:9090",
		HeartbeatInterval: 30,
		VectorBinPath:     defaultVectorBinPath(),
		VectorConfPath:    defaultVectorConfPath(),
		MaxMemoryMB:       512,
		OS:                runtime.GOOS,
	}
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

// IsWindows 判断是否运行在 Windows 上
func (c *Config) IsWindows() bool {
	return c.OS == "windows"
}

// IsLinux 判断是否运行在 Linux 上
func (c *Config) IsLinux() bool {
	return c.OS == "linux"
}
