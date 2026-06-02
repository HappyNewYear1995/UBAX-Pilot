package comm

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ubax/ubax-pilot/pkg/logger"
)

// ConfigRenderer 将服务端下发的抽象规则翻译为 Vector 配置文件
type ConfigRenderer struct {
	configPath string
}

// NewConfigRenderer 创建一个新的配置渲染器
func NewConfigRenderer(configPath string) *ConfigRenderer {
	return &ConfigRenderer{
		configPath: configPath,
	}
}

// Render 根据服务端规则生成 Vector 配置文件
func (cr *ConfigRenderer) Render(rules []byte) error {
	content := cr.renderVectorConfig(rules)

	if err := os.WriteFile(cr.configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	logger.Info("配置已渲染到:", cr.configPath)
	return nil
}

// renderVectorConfig 生成 Vector YAML 配置
func (cr *ConfigRenderer) renderVectorConfig(rules []byte) string {
	if len(rules) == 0 {
		return ""
	}

	// rules 是 JSON 编码的字符串，需要反序列化
	var yamlContent string
	if err := json.Unmarshal(rules, &yamlContent); err != nil {
		// 如果不是 JSON 字符串，尝试直接使用原始内容
		logger.Warn("配置内容不是 JSON 字符串格式，使用原始内容")
		return string(rules)
	}

	return yamlContent
}
