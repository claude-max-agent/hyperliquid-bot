package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents application configuration
type Config struct {
	App      AppConfig      `yaml:"app"`
	Exchange ExchangeConfig `yaml:"exchange"`
	Strategy StrategyConfig `yaml:"strategy"`
	Risk     RiskConfig     `yaml:"risk"`
	Log      LogConfig      `yaml:"log"`
}

// AppConfig represents application settings
type AppConfig struct {
	Name        string        `yaml:"name"`
	Environment string        `yaml:"environment"`
	Debug       bool          `yaml:"debug"`
	GracePeriod time.Duration `yaml:"grace_period"`
}

// ExchangeConfig represents exchange connection settings
type ExchangeConfig struct {
	Name       string `yaml:"name"`
	BaseURL    string `yaml:"base_url"`
	WSURL      string `yaml:"ws_url"`
	APIKey     string `yaml:"api_key"`
	APISecret  string `yaml:"api_secret"`
	Testnet    bool   `yaml:"testnet"`
	RateLimit  int    `yaml:"rate_limit"`
}

// StrategyConfig represents strategy settings
type StrategyConfig struct {
	Name   string                 `yaml:"name"`
	Symbol string                 `yaml:"symbol"`
	Params map[string]interface{} `yaml:"params"`
}

// RiskConfig represents risk management settings
type RiskConfig struct {
	MaxPositionSize float64 `yaml:"max_position_size"`
	MaxLeverage     float64 `yaml:"max_leverage"`
	MaxDrawdown     float64 `yaml:"max_drawdown"`
	DailyLossLimit  float64 `yaml:"daily_loss_limit"`
}

// LogConfig represents logging settings
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Load loads configuration from YAML file with env overrides
func Load(path string) (*Config, error) {
	cfg := &Config{}

	// Load from YAML file
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.loadEnvOverrides()

	// Validate
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// loadEnvOverrides overrides config with environment variables
func (c *Config) loadEnvOverrides() {
	// Exchange settings
	if v := os.Getenv("EXCHANGE_API_KEY"); v != "" {
		c.Exchange.APIKey = v
	}
	if v := os.Getenv("EXCHANGE_API_SECRET"); v != "" {
		c.Exchange.APISecret = v
	}
	if v := os.Getenv("EXCHANGE_BASE_URL"); v != "" {
		c.Exchange.BaseURL = v
	}
	if v := os.Getenv("EXCHANGE_WS_URL"); v != "" {
		c.Exchange.WSURL = v
	}
	if v := os.Getenv("EXCHANGE_TESTNET"); v != "" {
		c.Exchange.Testnet = v == "true" || v == "1"
	}

	// App settings
	if v := os.Getenv("APP_ENVIRONMENT"); v != "" {
		c.App.Environment = v
	}
	if v := os.Getenv("APP_DEBUG"); v != "" {
		c.App.Debug = v == "true" || v == "1"
	}

	// Log settings
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}

	// Risk settings
	if v := os.Getenv("RISK_MAX_POSITION_SIZE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Risk.MaxPositionSize = f
		}
	}
	if v := os.Getenv("RISK_MAX_LEVERAGE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Risk.MaxLeverage = f
		}
	}
}

// validate validates configuration
func (c *Config) validate() error {
	if c.Exchange.APIKey == "" {
		return fmt.Errorf("exchange.api_key is required")
	}
	if c.Exchange.APISecret == "" {
		return fmt.Errorf("exchange.api_secret is required")
	}
	if c.Strategy.Symbol == "" {
		return fmt.Errorf("strategy.symbol is required")
	}
	if c.Risk.MaxLeverage <= 0 {
		c.Risk.MaxLeverage = 1.0 // default
	}
	return nil
}
