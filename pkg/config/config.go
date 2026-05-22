package config

import (
	"runtime"
)

// Config holds the global configuration for UBAX-Pilot
type Config struct {
	// Pilot general settings
	PilotName string `yaml:"pilot_name"`
	Version   string `yaml:"version"`

	// Server connection settings
	ServerEndpoint string `yaml:"server_endpoint"`
	HeartbeatInterval int `yaml:"heartbeat_interval_seconds"`

	// Collector settings
	CollectorType     string `yaml:"collector_type"` // "vector" or "filebeat"
	CollectorBinPath  string `yaml:"collector_bin_path"`
	CollectorConfPath string `yaml:"collector_conf_path"`

	// Resource limits
	MaxCPUPercent   float64 `yaml:"max_cpu_percent"`
	MaxMemoryMB     int64   `yaml:"max_memory_mb"`

	// Buffer settings
	BufferDir       string `yaml:"buffer_dir"`
	MaxBufferSizeMB int64  `yaml:"max_buffer_size_mb"`

	// Platform-specific
	OS string `yaml:"-"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		PilotName:         "ubax-pilot",
		Version:           "0.1.0",
		ServerEndpoint:    "localhost:9090",
		HeartbeatInterval: 30,
		CollectorType:     "vector",
		CollectorBinPath:  defaultCollectorBinPath(),
		CollectorConfPath: defaultCollectorConfPath(),
		MaxCPUPercent:     80.0,
		MaxMemoryMB:       512,
		BufferDir:         defaultBufferDir(),
		MaxBufferSizeMB:   1024,
		OS:                runtime.GOOS,
	}
}

func defaultCollectorBinPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Program Files\ubax-pilot\vector\vector.exe`
	}
	return "/usr/local/bin/vector"
}

func defaultCollectorConfPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Program Files\ubax-pilot\config\vector.toml`
	}
	return "/etc/ubax-pilot/vector.toml"
}

func defaultBufferDir() string {
	if runtime.GOOS == "windows" {
		return `C:\Program Files\ubax-pilot\buffer`
	}
	return "/var/lib/ubax-pilot/buffer"
}

// LoadConfig loads configuration from file, falling back to defaults
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	// TODO: implement file-based config loading (yaml/toml)
	_ = path

	return cfg, nil
}

// GetServiceName returns the platform-specific service name
func (c *Config) GetServiceName() string {
	return c.PilotName
}

// IsWindows returns true if running on Windows
func (c *Config) IsWindows() bool {
	return c.OS == "windows"
}

// IsLinux returns true if running on Linux
func (c *Config) IsLinux() bool {
	return c.OS == "linux"
}
