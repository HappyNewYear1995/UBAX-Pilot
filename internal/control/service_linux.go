//go:build linux

package control

import (
	"fmt"
	"os"
	"os/exec"
)

const systemdServiceTemplate = `[Unit]
Description=UBAX-Pilot Data Collection Agent
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

// Install registers the pilot as a systemd service
func (sa *ServiceAdapter) Install() error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	serviceContent := fmt.Sprintf(systemdServiceTemplate, binPath)
	servicePath := "/etc/systemd/system/ubax-pilot.service"

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	cmd = exec.Command("systemctl", "enable", "ubax-pilot")
	return cmd.Run()
}

// Start starts the systemd service
func (sa *ServiceAdapter) Start() error {
	cmd := exec.Command("systemctl", "start", "ubax-pilot")
	return cmd.Run()
}

// Stop stops the systemd service
func (sa *ServiceAdapter) Stop() error {
	cmd := exec.Command("systemctl", "stop", "ubax-pilot")
	return cmd.Run()
}

// Uninstall removes the systemd service
func (sa *ServiceAdapter) Uninstall() error {
	exec.Command("systemctl", "stop", "ubax-pilot").Run()
	exec.Command("systemctl", "disable", "ubax-pilot").Run()

	servicePath := "/etc/systemd/system/ubax-pilot.service"
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// GetConfigDir returns the default config directory for Linux
func GetConfigDir() string {
	return "/etc/ubax-pilot"
}

// GetDataDir returns the default data directory for Linux
func GetDataDir() string {
	return "/var/lib/ubax-pilot"
}

// GetLogDir returns the default log directory for Linux
func GetLogDir() string {
	return "/var/log/ubax-pilot"
}
