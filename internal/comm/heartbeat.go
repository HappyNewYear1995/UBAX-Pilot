package comm

import (
	"context"
	"time"

	"github.com/ubax/ubax-pilot/pkg/config"
	"github.com/ubax/ubax-pilot/pkg/logger"
)

// HeartbeatReporter periodically sends health status to the server
type HeartbeatReporter struct {
	cfg      *config.Config
	interval time.Duration
	stopCh   chan struct{}
}

// HeartbeatPayload contains the heartbeat data sent to server
type HeartbeatPayload struct {
	Version     string    `json:"version"`
	Hostname    string    `json:"hostname"`
	Timestamp   time.Time `json:"timestamp"`
	Healthy     bool      `json:"healthy"`
	CollectorStatus string `json:"collector_status"`
	OS          string    `json:"os"`
}

// NewHeartbeatReporter creates a new heartbeat reporter
func NewHeartbeatReporter(cfg *config.Config) *HeartbeatReporter {
	return &HeartbeatReporter{
		cfg:      cfg,
		interval: time.Duration(cfg.HeartbeatInterval) * time.Second,
		stopCh:   make(chan struct{}),
	}
}

// Start begins periodic heartbeat reporting
func (hr *HeartbeatReporter) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(hr.interval)
		defer ticker.Stop()

		// Send initial heartbeat
		hr.sendHeartbeat(ctx)

		for {
			select {
			case <-ticker.C:
				hr.sendHeartbeat(ctx)
			case <-hr.stopCh:
				return
			}
		}
	}()
}

// Stop stops the heartbeat reporter
func (hr *HeartbeatReporter) Stop() {
	close(hr.stopCh)
}

func (hr *HeartbeatReporter) sendHeartbeat(ctx context.Context) {
	payload := &HeartbeatPayload{
		Version:   hr.cfg.Version,
		Timestamp: time.Now(),
		Healthy:   true,
		OS:        hr.cfg.OS,
	}

	// TODO: Send payload to server via HTTP/gRPC
	logger.Debug("Heartbeat sent:", payload.Version, "healthy:", payload.Healthy)
}
