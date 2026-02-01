package risk

import (
	"sync"
	"time"
)

// Config holds risk management configuration
type Config struct {
	MaxPositionSize     float64
	MaxDailyLoss        float64
	MaxConsecutiveLoss  int
	CooldownDuration    time.Duration
}

// DefaultConfig returns default risk configuration
func DefaultConfig() *Config {
	return &Config{
		MaxPositionSize:    1.0,
		MaxDailyLoss:       0.05, // 5%
		MaxConsecutiveLoss: 3,
		CooldownDuration:   5 * time.Minute,
	}
}

// CheckResult represents the result of a risk check
type CheckResult struct {
	Allowed bool
	Reason  string
}

// Checker performs risk checks before order execution
type Checker struct {
	config *Config

	mu               sync.RWMutex
	dailyPnL         float64
	consecutiveLoss  int
	cooldownUntil    time.Time
	halted           bool
	haltReason       string
}

// NewChecker creates a new risk checker
func NewChecker(cfg *Config) *Checker {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Checker{
		config: cfg,
	}
}

// CanTrade checks if trading is allowed
func (c *Checker) CanTrade() CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.halted {
		return CheckResult{Allowed: false, Reason: "trading halted: " + c.haltReason}
	}

	if time.Now().Before(c.cooldownUntil) {
		return CheckResult{Allowed: false, Reason: "in cooldown until " + c.cooldownUntil.Format(time.RFC3339)}
	}

	if c.dailyPnL < -c.config.MaxDailyLoss {
		return CheckResult{Allowed: false, Reason: "daily loss limit exceeded"}
	}

	return CheckResult{Allowed: true}
}

// CheckPositionSize validates position size
func (c *Checker) CheckPositionSize(size float64) CheckResult {
	if size > c.config.MaxPositionSize {
		return CheckResult{
			Allowed: false,
			Reason:  "position size exceeds maximum",
		}
	}
	return CheckResult{Allowed: true}
}

// RecordTrade records a trade result
func (c *Checker) RecordTrade(pnl float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dailyPnL += pnl

	if pnl < 0 {
		c.consecutiveLoss++
		if c.consecutiveLoss >= c.config.MaxConsecutiveLoss {
			c.cooldownUntil = time.Now().Add(c.config.CooldownDuration)
			c.consecutiveLoss = 0
		}
	} else {
		c.consecutiveLoss = 0
	}
}

// Halt stops trading
func (c *Checker) Halt(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.halted = true
	c.haltReason = reason
}

// Resume resumes trading
func (c *Checker) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.halted = false
	c.haltReason = ""
	c.consecutiveLoss = 0
}

// ResetDaily resets daily statistics
func (c *Checker) ResetDaily() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dailyPnL = 0
}

// Status returns current risk status
func (c *Checker) Status() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"halted":           c.halted,
		"halt_reason":      c.haltReason,
		"daily_pnl":        c.dailyPnL,
		"consecutive_loss": c.consecutiveLoss,
		"in_cooldown":      time.Now().Before(c.cooldownUntil),
		"cooldown_until":   c.cooldownUntil,
	}
}
