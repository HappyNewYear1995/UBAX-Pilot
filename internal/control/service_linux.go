//go:build linux

package control

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func init() {
	// Override platform detection for Linux builds
}

func detectPlatform() string {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return "systemd"
	}
	return "linux-nosystemd"
}

func (sa *ServiceAdapter) installSystemd() error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	serviceContent := fmt.Sprintf(systemdServiceTemplate, binPath)
	servicePath := "/etc/systemd/system/ubax-pilot.service"

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd daemon
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	cmd = exec.Command("systemctl", "enable", "ubax-pilot")
	return cmd.Run()
}

func (sa *ServiceAdapter) startSystemd() error {
	cmd := exec.Command("systemctl", "start", "ubax-pilot")
	return cmd.Run()
}

func (sa *ServiceAdapter) stopSystemd() error {
	cmd := exec.Command("systemctl", "stop", "ubax-pilot")
	return cmd.Run()
}

func (sa *ServiceAdapter) uninstallSystemd() error {
	// Stop and disable first
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
