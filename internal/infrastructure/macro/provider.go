package macro

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

// Provider aggregates macro data sources
type Provider struct {
	fedWatch         *FedWatchClient
	tradingEconomics *TradingEconomicsClient

	mu             sync.RWMutex
	running        bool
	signalHandlers []func(*entity.MacroSignal)

	// Cached data
	cachedFedWatch *entity.FedWatchData
	cachedMacro    *entity.MacroSignal
}

// Config holds macro provider configuration
type Config struct {
	FedWatchAPIKey         string
	TradingEconomicsAPIKey string
}

// NewProvider creates a new macro provider
func NewProvider(cfg Config) *Provider {
	var fw *FedWatchClient
	var te *TradingEconomicsClient

	if cfg.FedWatchAPIKey != "" {
		fw = NewFedWatchClient(cfg.FedWatchAPIKey)
	}
	if cfg.TradingEconomicsAPIKey != "" {
		te = NewTradingEconomicsClient(cfg.TradingEconomicsAPIKey)
	}

	return &Provider{
		fedWatch:         fw,
		tradingEconomics: te,
		signalHandlers:   make([]func(*entity.MacroSignal), 0),
	}
}

// Start starts macro data collection
func (p *Provider) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.mu.Unlock()

	// Connect FedWatch
	if p.fedWatch != nil {
		if err := p.fedWatch.Connect(ctx); err != nil {
			// Log warning but continue
		}
	}

	// Connect Trading Economics
	if p.tradingEconomics != nil {
		if err := p.tradingEconomics.Connect(ctx); err != nil {
			// Log warning but continue
		}
	}

	// Start background data collection
	go p.collectData(ctx)

	// Subscribe to FedWatch updates
	if p.fedWatch != nil {
		p.fedWatch.SubscribeFedWatch(ctx, func(data *entity.FedWatchData) {
			p.mu.Lock()
			p.cachedFedWatch = data
			p.mu.Unlock()
			p.broadcastSignal()
		})
	}

	// Subscribe to indicator updates
	if p.tradingEconomics != nil {
		p.tradingEconomics.SubscribeIndicators(ctx, func(signal *entity.MacroSignal) {
			p.mu.Lock()
			// Merge with FedWatch data
			if p.cachedFedWatch != nil {
				signal.FedWatch = p.cachedFedWatch
			}
			p.cachedMacro = signal
			p.mu.Unlock()
			p.broadcastSignal()
		})
	}

	return nil
}

// Stop stops macro data collection
func (p *Provider) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	if p.fedWatch != nil {
		p.fedWatch.Disconnect(ctx)
	}
	if p.tradingEconomics != nil {
		p.tradingEconomics.Disconnect(ctx)
	}

	return nil
}

// collectData periodically collects macro data
func (p *Provider) collectData(ctx context.Context) {
	// Initial collection
	p.refreshData(ctx)

	// Periodic refresh
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.mu.RLock()
			running := p.running
			p.mu.RUnlock()

			if !running {
				return
			}

			p.refreshData(ctx)
		}
	}
}

// refreshData fetches fresh data from all sources
func (p *Provider) refreshData(ctx context.Context) {
	signal := &entity.MacroSignal{
		Timestamp: time.Now(),
	}

	// Get FedWatch data
	if p.fedWatch != nil {
		if data, err := p.fedWatch.GetFedWatchData(ctx); err == nil {
			signal.FedWatch = data
			p.mu.Lock()
			p.cachedFedWatch = data
			p.mu.Unlock()
		}
	}

	// Get economic indicators
	if p.tradingEconomics != nil {
		if cpi, err := p.tradingEconomics.GetUSInflation(ctx); err == nil {
			signal.CPI = cpi
		}
		if gdp, err := p.tradingEconomics.GetUSGDP(ctx); err == nil {
			signal.GDP = gdp
		}
		if unemp, err := p.tradingEconomics.GetUSUnemployment(ctx); err == nil {
			signal.Unemployment = unemp
		}
		if pce, err := p.tradingEconomics.GetUSPCE(ctx); err == nil {
			signal.PCE = pce
		}
		if events, err := p.tradingEconomics.GetHighImpactEvents(ctx, 7); err == nil {
			signal.UpcomingEvents = events
		}
	}

	signal.AnalyzeMacroSignal()

	p.mu.Lock()
	p.cachedMacro = signal
	p.mu.Unlock()

	p.broadcastSignal()
}

// GetMacroSignal returns the current macro signal
func (p *Provider) GetMacroSignal(ctx context.Context) (*entity.MacroSignal, error) {
	p.mu.RLock()
	cached := p.cachedMacro
	p.mu.RUnlock()

	if cached != nil && time.Since(cached.Timestamp) < 15*time.Minute {
		return cached, nil
	}

	// Refresh if stale
	signal := &entity.MacroSignal{
		Timestamp: time.Now(),
	}

	if p.fedWatch != nil {
		if data, err := p.fedWatch.GetFedWatchData(ctx); err == nil {
			signal.FedWatch = data
		}
	}

	if p.tradingEconomics != nil {
		if cpi, err := p.tradingEconomics.GetUSInflation(ctx); err == nil {
			signal.CPI = cpi
		}
		if gdp, err := p.tradingEconomics.GetUSGDP(ctx); err == nil {
			signal.GDP = gdp
		}
		if unemp, err := p.tradingEconomics.GetUSUnemployment(ctx); err == nil {
			signal.Unemployment = unemp
		}
	}

	signal.AnalyzeMacroSignal()

	p.mu.Lock()
	p.cachedMacro = signal
	p.mu.Unlock()

	return signal, nil
}

// GetFedWatchData returns the current FedWatch data
func (p *Provider) GetFedWatchData(ctx context.Context) (*entity.FedWatchData, error) {
	if p.fedWatch == nil {
		return nil, nil
	}
	return p.fedWatch.GetFedWatchData(ctx)
}

// SubscribeSignals subscribes to macro signal updates
func (p *Provider) SubscribeSignals(ctx context.Context, handler func(*entity.MacroSignal)) error {
	p.mu.Lock()
	p.signalHandlers = append(p.signalHandlers, handler)
	p.mu.Unlock()
	return nil
}

// broadcastSignal broadcasts the current macro signal
func (p *Provider) broadcastSignal() {
	p.mu.RLock()
	signal := p.cachedMacro
	handlers := make([]func(*entity.MacroSignal), len(p.signalHandlers))
	copy(handlers, p.signalHandlers)
	p.mu.RUnlock()

	if signal == nil {
		return
	}

	for _, handler := range handlers {
		handler(signal)
	}
}

// GetMacroSummary returns a human-readable summary
func GetMacroSummary(signal *entity.MacroSignal) string {
	if signal == nil {
		return "Macro: No data available"
	}

	summary := "Macro Signal: " + string(signal.Bias)
	summary += " (Strength: " + formatPercent(signal.Strength) + ")\n"

	if signal.FedWatch != nil && signal.FedWatch.NextMeeting != nil {
		m := signal.FedWatch.NextMeeting
		summary += "  Fed: " + m.MeetingDate.Format("Jan 2") + " - "
		summary += "Cut " + formatPercent(m.CutProb) + " | Hold " + formatPercent(m.HoldProb) + " | Hike " + formatPercent(m.HikeProb) + "\n"
	}

	if signal.CPI != nil {
		summary += "  CPI: " + formatFloat(signal.CPI.Value) + "% (prev: " + formatFloat(signal.CPI.Previous) + "%)\n"
	}

	if signal.Unemployment != nil {
		summary += "  Unemployment: " + formatFloat(signal.Unemployment.Value) + "%\n"
	}

	if len(signal.UpcomingEvents) > 0 {
		summary += "  Upcoming: " + signal.UpcomingEvents[0].Event + " (" + signal.UpcomingEvents[0].Date.Format("Jan 2") + ")"
	}

	return summary
}

func formatPercent(v float64) string {
	return formatFloat(v*100) + "%"
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.1f", v)
}
