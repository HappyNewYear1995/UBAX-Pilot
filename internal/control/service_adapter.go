package control

// ServiceAdapter registers and manages the pilot as a system service
type ServiceAdapter struct {
	platform string
}

// NewServiceAdapter creates a platform-specific service adapter
func NewServiceAdapter() *ServiceAdapter {
	return &ServiceAdapter{
		platform: detectPlatform(),
	}
}

// Install registers the pilot as a system service
func (sa *ServiceAdapter) Install() error {
	switch sa.platform {
	case "systemd":
		return sa.installSystemd()
	case "windows":
		return sa.installWindowsService()
	default:
		return nil
	}
}

// Start starts the system service
func (sa *ServiceAdapter) Start() error {
	switch sa.platform {
	case "systemd":
		return sa.startSystemd()
	case "windows":
		return sa.startWindowsService()
	default:
		return nil
	}
}

// Stop stops the system service
func (sa *ServiceAdapter) Stop() error {
	switch sa.platform {
	case "systemd":
		return sa.stopSystemd()
	case "windows":
		return sa.stopWindowsService()
	default:
		return nil
	}
}

// Uninstall removes the system service
func (sa *ServiceAdapter) Uninstall() error {
	switch sa.platform {
	case "systemd":
		return sa.uninstallSystemd()
	case "windows":
		return sa.uninstallWindowsService()
	default:
		return nil
	}
}

func detectPlatform() string {
	// Platform detection is handled in platform-specific files
	return "unknown"
}
