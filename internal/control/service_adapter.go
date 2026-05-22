package control

// ServiceAdapter registers and manages the pilot as a system service
type ServiceAdapter struct{}

// NewServiceAdapter creates a platform-specific service adapter
func NewServiceAdapter() *ServiceAdapter {
	return &ServiceAdapter{}
}
