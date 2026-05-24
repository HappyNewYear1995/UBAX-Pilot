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

	// Vector 头部注释
	sb.WriteString(`#                                    __   __  __
#                                    \ \ / / / /
#                                     \ V / / /
#                                      \_/  \/
#
#                                    V E C T O R
#                                   Configuration
#
# ------------------------------------------------------------------------------
# Website: ` + "`https://vector.dev`" + `
# Docs: ` + "`https://vector.dev/docs`" + `
# Chat: ` + "`https://chat.vector.dev`" + `
# ------------------------------------------------------------------------------
#
# Change this to use a non-default directory for Vector data storage:
# data_dir: "/var/lib/vector"

`)

	// 如果 rules 为空，生成默认配置
	if len(rules) == 0 {
		sb.WriteString(`# Random Syslog-formatted logs
sources:
  dummy_logs:
    type: "demo_logs"
    format: "syslog"
    interval: 1

# Parse Syslog logs
# See the Vector Remap Language reference for more info: ` + "`https://vrl.dev`" + `
transforms:
  parse_logs:
    type: "remap"
    inputs: ["dummy_logs"]
    source: |
      . = parse_syslog!(string!(.message))

# Print parsed logs to stdout
sinks:
  print:
    type: "console"
    inputs: ["parse_logs"]
    encoding:
      codec: "json"
      json:
        pretty: true

# Vector's API (disabled by default)
# Uncomment to try it out with the ` + "`vector top`" + ` command
# api:
#   enabled: true
#   address: "127.0.0.1:8686"
`)
	} else {
		// TODO: 解析服务端规则并生成对应的 sources/transforms/sinks
		sb.WriteString(string(rules))
	}

	return sb.String()
}

// TriggerHotReload 通知 Vector 重新加载配置
func (cr *ConfigRenderer) TriggerHotReload() error {
	// TODO: 向 Vector 进程发送 SIGHUP 信号或调用重载 API
	logger.Info("正在触发 Vector 热重载")
	return nil
}
