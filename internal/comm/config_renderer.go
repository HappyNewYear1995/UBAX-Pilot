package comm

import (
	"fmt"
	"os"
	"strings"

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

// renderVectorConfig 生成 vector.yaml 配置
func (cr *ConfigRenderer) renderVectorConfig(rules []byte) string {
	var sb strings.Builder
	// 如果 rules 不为空，生成配置
	if len(rules) != 0 {
		sb.WriteString(string(rules))
	}

	return sb.String()
}
