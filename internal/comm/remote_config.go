package comm

import (
	"context"
	"fmt"
	"time"

	"github.com/ubax/ubax-pilot/pkg/config"
	"github.com/ubax/ubax-pilot/pkg/logger"
)

// RemoteConfigClient handles communication with the UBAX server
type RemoteConfigClient struct {
	cfg          *config.Config
	endpoint     string
	pollInterval time.Duration
	stopCh       chan struct{}
}

// NewRemoteConfigClient creates a new remote config client
func NewRemoteConfigClient(cfg *config.Config) *RemoteConfigClient {
	return &RemoteConfigClient{
		cfg:          cfg,
		endpoint:     cfg.ServerEndpoint,
		pollInterval: time.Duration(cfg.HeartbeatInterval) * time.Second,
		stopCh:       make(chan struct{}),
	}
}

// FetchConfig retrieves the latest configuration from the server
func (rc *RemoteConfigClient) FetchConfig(ctx context.Context) ([]byte, error) {
	// TODO: Implement actual gRPC/HTTP call to fetch config
	logger.Info("Fetching config from:", rc.endpoint)
	return nil, fmt.Errorf("not implemented")
}

// StartPolling begins periodic config polling
func (rc *RemoteConfigClient) StartPolling(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(rc.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if _, err := rc.FetchConfig(ctx); err != nil {
					logger.Error("Failed to fetch config:", err)
				}
			case <-rc.stopCh:
				return
			}
		}
	}()
}

// Stop stops the config polling
func (rc *RemoteConfigClient) Stop() {
	close(rc.stopCh)
}
