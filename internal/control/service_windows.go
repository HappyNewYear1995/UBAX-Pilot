//go:build windows

package control

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func init() {
	// Override platform detection for Windows builds
}

func detectPlatform() string {
	return "windows"
}

func (sa *ServiceAdapter) installWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	conf := mgr.Config{
		DisplayName:  "UBAX-Pilot Data Collection Agent",
		Description:  "UBAX-Pilot is a cross-platform data collection agent",
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	}

	s, err := m.CreateService("ubax-pilot", binPath, conf)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	return nil
}

func (sa *ServiceAdapter) startWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("ubax-pilot")
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	return s.Start()
}

func (sa *ServiceAdapter) stopWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("ubax-pilot")
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	return err
}

func (sa *ServiceAdapter) uninstallWindowsService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("ubax-pilot")
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	s.Control(svc.Stop)
	return s.Delete()
}

// RunAsService runs the pilot as a Windows service
func RunAsService(handler svc.Handler) error {
	return svc.Run("ubax-pilot", handler)
}

// GetConfigDir returns the default config directory for Windows
func GetConfigDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "ubax-pilot", "config")
}

// GetDataDir returns the default data directory for Windows
func GetDataDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "ubax-pilot", "data")
}

// GetLogDir returns the default log directory for Windows
func GetLogDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "ubax-pilot", "logs")
}
