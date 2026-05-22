package reliability

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ubax/ubax-pilot/pkg/logger"
)

// LocalBuffer provides persistent local storage for data when network is unavailable
type LocalBuffer struct {
	bufferDir     string
	maxSizeBytes  int64
	currentSize   int64
	mu            sync.Mutex
}

// NewLocalBuffer creates a new local buffer
func NewLocalBuffer(bufferDir string, maxSizeMB int64) *LocalBuffer {
	return &LocalBuffer{
		bufferDir:    bufferDir,
		maxSizeBytes: maxSizeMB * 1024 * 1024,
	}
}

// Init initializes the buffer directory
func (lb *LocalBuffer) Init() error {
	if err := os.MkdirAll(lb.bufferDir, 0755); err != nil {
		return fmt.Errorf("failed to create buffer directory: %w", err)
	}
	logger.Info("Local buffer initialized at:", lb.bufferDir)
	return nil
}

// Write stores data to the local buffer
func (lb *LocalBuffer) Write(data []byte) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.currentSize+int64(len(data)) > lb.maxSizeBytes {
		return fmt.Errorf("buffer size limit exceeded")
	}

	// Write to file-based queue
	filename := filepath.Join(lb.bufferDir, fmt.Sprintf("buf_%d.dat", len(data)))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write to buffer: %w", err)
	}

	lb.currentSize += int64(len(data))
	logger.Debug("Data buffered:", len(data), "bytes")
	return nil
}

// Read retrieves data from the local buffer for replay
func (lb *LocalBuffer) Read() ([]byte, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	entries, err := os.ReadDir(lb.bufferDir)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	// Read oldest file first
	filename := filepath.Join(lb.bufferDir, entries[0].Name())
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Remove after successful read
	os.Remove(filename)
	lb.currentSize -= int64(len(data))

	return data, nil
}

// Size returns the current buffer size in bytes
func (lb *LocalBuffer) Size() int64 {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.currentSize
}

// Clear removes all buffered data
func (lb *LocalBuffer) Clear() error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	entries, err := os.ReadDir(lb.bufferDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		os.Remove(filepath.Join(lb.bufferDir, entry.Name()))
	}

	lb.currentSize = 0
	logger.Info("Local buffer cleared")
	return nil
}
