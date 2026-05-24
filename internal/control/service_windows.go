//go:build windows

package control

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Install 将 Pilot 注册为 Windows 服务
func (sa *ServiceAdapter) Install() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	conf := mgr.Config{
		DisplayName:  "UBAX-Pilot 数据采集代理",
		Description:  "UBAX-Pilot 是一个跨平台数据采集代理",
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	}

	s, err := m.CreateService("ubax-pilot", binPath, conf)
	if err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}
	defer func(s *mgr.Service) {
		_ = s.Close()
	}(s)

	return nil
}

// Start 启动 Windows 服务
func (sa *ServiceAdapter) Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	s, err := m.OpenService("ubax-pilot")
	if err != nil {
		return fmt.Errorf("打开服务失败: %w", err)
	}
	defer func(s *mgr.Service) {
		_ = s.Close()
	}(s)

	return s.Start()
}

// Stop 停止 Windows 服务
func (sa *ServiceAdapter) Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	s, err := m.OpenService("ubax-pilot")
	if err != nil {
		return fmt.Errorf("打开服务失败: %w", err)
	}
	defer func(s *mgr.Service) {
		_ = s.Close()
	}(s)

	_, err = s.Control(svc.Stop)
	return err
}

// Uninstall 移除 Windows 服务
func (sa *ServiceAdapter) Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	s, err := m.OpenService("ubax-pilot")
	if err != nil {
		return fmt.Errorf("打开服务失败: %w", err)
	}
	defer func(s *mgr.Service) {
		_ = s.Close()
	}(s)

	_, _ = s.Control(svc.Stop)
	return s.Delete()
}

// RunAsService 以 Windows 服务方式运行 Pilot
func RunAsService(handler svc.Handler) error {
	return svc.Run("ubax-pilot", handler)
}

// GetConfigDir 返回 Windows 平台的默认配置目录
func GetConfigDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "ubax-pilot", "config")
}

// GetDataDir 返回 Windows 平台的默认数据目录
func GetDataDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "ubax-pilot", "data")
}

// GetLogDir 返回 Windows 平台的默认日志目录
func GetLogDir() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "ubax-pilot", "logs")
}
