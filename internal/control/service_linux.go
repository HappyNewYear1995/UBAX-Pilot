//go:build linux

package control

import (
	"fmt"
	"os"
	"os/exec"
)

const systemdServiceTemplate = `[Unit]
Description=UBAX-Pilot 数据采集代理
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`

// Install 将 Pilot 注册为 systemd 服务
func (sa *ServiceAdapter) Install() error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	serviceContent := fmt.Sprintf(systemdServiceTemplate, binPath)
	servicePath := "/etc/systemd/system/ubax-pilot.service"

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("写入服务文件失败: %w", err)
	}

	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("重新加载 systemd 失败: %w", err)
	}

	cmd = exec.Command("systemctl", "enable", "ubax-pilot")
	return cmd.Run()
}

// Start 启动 systemd 服务
func (sa *ServiceAdapter) Start() error {
	cmd := exec.Command("systemctl", "start", "ubax-pilot")
	return cmd.Run()
}

// Stop 停止 systemd 服务
func (sa *ServiceAdapter) Stop() error {
	cmd := exec.Command("systemctl", "stop", "ubax-pilot")
	return cmd.Run()
}

// Uninstall 移除 systemd 服务
func (sa *ServiceAdapter) Uninstall() error {
	exec.Command("systemctl", "stop", "ubax-pilot").Run()
	exec.Command("systemctl", "disable", "ubax-pilot").Run()

	servicePath := "/etc/systemd/system/ubax-pilot.service"
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("移除服务文件失败: %w", err)
	}

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// GetConfigDir 返回 Linux 平台的默认配置目录
func GetConfigDir() string {
	return "/etc/ubax-pilot"
}

// GetDataDir 返回 Linux 平台的默认数据目录
func GetDataDir() string {
	return "/var/lib/ubax-pilot"
}

// GetLogDir 返回 Linux 平台的默认日志目录
func GetLogDir() string {
	return "/var/log/ubax-pilot"
}
