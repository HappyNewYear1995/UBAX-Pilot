package reliability

import (
	"sync"
	"time"

	"github.com/ubax/ubax-pilot/pkg/logger"
)

// CircuitBreaker implements adaptive circuit breaking based on network conditions
type CircuitBreaker struct {
	mu              sync.Mutex
	state           string // "closed", "open", "half-open"
	failureCount    int
	successCount    int
	threshold       int
	timeout         time.Duration
	lastFailureTime time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     "closed",
		threshold: threshold,
		timeout:   timeout,
	}
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case "closed":
		return true
	case "open":
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = "half-open"
			logger.Info("Circuit breaker: open -> half-open")
			return true
		}
		return false
	case "half-open":
		return true
	default:
		return true
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "half-open" {
		cb.successCount++
		if cb.successCount >= 3 {
			cb.state = "closed"
			cb.failureCount = 0
			cb.successCount = 0
			logger.Info("Circuit breaker: half-open -> closed")
		}
	} else {
		cb.failureCount = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == "half-open" {
		cb.state = "open"
		cb.successCount = 0
		logger.Info("Circuit breaker: half-open -> open")
	} else if cb.failureCount >= cb.threshold {
		cb.state = "open"
		logger.Warn("Circuit breaker: closed -> open (failures:", cb.failureCount, ")")
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// RateLimiter implements adaptive rate limiting based on system load
type RateLimiter struct {
	mu            sync.Mutex
	currentRate   int
	maxRate       int
	minRate       int
	lastAdjustTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRate, minRate int) *RateLimiter {
	return &RateLimiter{
		currentRate: maxRate,
		maxRate:     maxRate,
		minRate:     minRate,
	}
}

// Allow checks if a request is allowed under current rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.currentRate <= 0 {
		return false
	}

	rl.currentRate--
	return true
}

// AdjustRate dynamically adjusts the rate based on conditions
func (rl *RateLimiter) AdjustRate(increase bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastAdjustTime) < time.Second {
		return // Prevent too frequent adjustments
	}
	rl.lastAdjustTime = now

	if increase && rl.currentRate < rl.maxRate {
		rl.currentRate += rl.currentRate / 10 + 1
		if rl.currentRate > rl.maxRate {
			rl.currentRate = rl.maxRate
		}
		logger.Debug("Rate limiter: increased to", rl.currentRate)
	} else if !increase && rl.currentRate > rl.minRate {
		rl.currentRate -= rl.currentRate / 10 + 1
		if rl.currentRate < rl.minRate {
			rl.currentRate = rl.minRate
		}
		logger.Debug("Rate limiter: decreased to", rl.currentRate)
	}
}

// GetCurrentRate returns the current allowed request rate
func (rl *RateLimiter) GetCurrentRate() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.currentRate
}
