package config

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

// BuiltInConfig 内置配置（版本号等）
type BuiltInConfig struct {
	Version   string `yaml:"version"`
	BuildTime string `yaml:"build_time"`
	GitCommit string `yaml:"git_commit"`
}

//go:embed built_in.yaml
var builtInYAML []byte

// GetBuiltInConfig 解析并返回内置配置
func GetBuiltInConfig() (*BuiltInConfig, error) {
	var cfg BuiltInConfig
	if err := yaml.Unmarshal(builtInYAML, &cfg); err != nil {
		return nil, fmt.Errorf("解析内置配置失败: %w", err)
	}
	return &cfg, nil
}

// GetVersion 返回当前版本号
func GetVersion() string {
	cfg, err := GetBuiltInConfig()
	if err != nil {
		return "unknown"
	}
	return cfg.Version
}
