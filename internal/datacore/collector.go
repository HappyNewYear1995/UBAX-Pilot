package datacore

import (
	"context"
	"fmt"

	"github.com/ubax/ubax-pilot/pkg/config"
	"github.com/ubax/ubax-pilot/pkg/logger"
)

// Collector defines the interface for data collection backends
type Collector interface {
	// Start begins data collection
	Start(ctx context.Context) error
	// Stop stops data collection
	Stop() error
	// ReloadConfig triggers a configuration reload
	ReloadConfig() error
	// HealthCheck returns the collector health status
	HealthCheck() bool
}

// DataCore wraps the underlying collector (Vector/Filebeat)
type DataCore struct {
	cfg       *config.Config
	collector Collector
}

// NewDataCore creates a new data core instance
func NewDataCore(cfg *config.Config) (*DataCore, error) {
	var collector Collector

	switch cfg.CollectorType {
	case "vector":
		collector = NewVectorCollector(cfg)
	case "filebeat":
		collector = NewFilebeatCollector(cfg)
	default:
		return nil, fmt.Errorf("unsupported collector type: %s", cfg.CollectorType)
	}

	return &DataCore{
		cfg:       cfg,
		collector: collector,
	}, nil
}

// Start begins data collection
func (dc *DataCore) Start(ctx context.Context) error {
	logger.Info("Starting data core with collector:", dc.cfg.CollectorType)
	return dc.collector.Start(ctx)
}

// Stop stops data collection
func (dc *DataCore) Stop() error {
	return dc.collector.Stop()
}

// ReloadConfig reloads the collector configuration
func (dc *DataCore) ReloadConfig() error {
	return dc.collector.ReloadConfig()
}

// HealthCheck returns whether the collector is healthy
func (dc *DataCore) HealthCheck() bool {
	return dc.collector.HealthCheck()
}

// VectorCollector implements Collector for Vector
type VectorCollector struct {
	cfg *config.Config
}

// NewVectorCollector creates a new Vector collector wrapper
func NewVectorCollector(cfg *config.Config) *VectorCollector {
	return &VectorCollector{cfg: cfg}
}

func (vc *VectorCollector) Start(ctx context.Context) error {
	logger.Info("Starting Vector collector...")
	// Vector will be managed by the ProcessManager in control layer
	return nil
}

func (vc *VectorCollector) Stop() error {
	logger.Info("Stopping Vector collector...")
	return nil
}

func (vc *VectorCollector) ReloadConfig() error {
	logger.Info("Reloading Vector config...")
	// Vector supports hot reload via SIGHUP or API
	return nil
}

func (vc *VectorCollector) HealthCheck() bool {
	return true
}

// FilebeatCollector implements Collector for Filebeat
type FilebeatCollector struct {
	cfg *config.Config
}

// NewFilebeatCollector creates a new Filebeat collector wrapper
func NewFilebeatCollector(cfg *config.Config) *FilebeatCollector {
	return &FilebeatCollector{cfg: cfg}
}

func (fc *FilebeatCollector) Start(ctx context.Context) error {
	logger.Info("Starting Filebeat collector...")
	return nil
}

func (fc *FilebeatCollector) Stop() error {
	logger.Info("Stopping Filebeat collector...")
	return nil
}

func (fc *FilebeatCollector) ReloadConfig() error {
	logger.Info("Reloading Filebeat config...")
	return nil
}

func (fc *FilebeatCollector) HealthCheck() bool {
	return true
}
