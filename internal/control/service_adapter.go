package control

// ServiceAdapter 注册并管理 Pilot 作为系统服务
type ServiceAdapter struct{}

// NewServiceAdapter 创建平台特定的服务适配器
func NewServiceAdapter() *ServiceAdapter {
	return &ServiceAdapter{}
}
