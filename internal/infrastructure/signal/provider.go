package signal

import (
	"context"
	"sync"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/infrastructure/coinglass"
	"github.com/zono819/hyperliquid-bot/internal/infrastructure/lunarcrush"
	"github.com/zono819/hyperliquid-bot/internal/infrastructure/whalealert"
)

// Provider aggregates multiple data sources for market signals
type Provider struct {
	coinglass  *coinglass.Client
	whalealert *whalealert.Client
	lunarcrush *lunarcrush.Client

	mu             sync.RWMutex
	running        bool
	symbols        []string
	signalHandlers []func(*entity.MarketSignal)

	// Cached data
	recentWhaleAlerts  map[string][]*entity.WhaleAlert     // symbol -> alerts
	recentLiquidations map[string][]*entity.Liquidation    // symbol -> liquidations
	recentSentiment    map[string]*entity.SocialSentiment  // symbol -> sentiment
}

// Config holds provider configuration
type Config struct {
	CoinGlassAPIKey   string
	WhaleAlertAPIKey  string
	WhaleMinValue     float64
	LunarCrushAPIKey  string
	Symbols           []string
}

// NewProvider creates a new signal provider
func NewProvider(cfg Config) *Provider {
	var cg *coinglass.Client
	var wa *whalealert.Client
	var lc *lunarcrush.Client

	if cfg.CoinGlassAPIKey != "" {
		cg = coinglass.NewClient(cfg.CoinGlassAPIKey)
	}
	if cfg.WhaleAlertAPIKey != "" {
		wa = whalealert.NewClient(cfg.WhaleAlertAPIKey, cfg.WhaleMinValue)
	}
	if cfg.LunarCrushAPIKey != "" {
		lc = lunarcrush.NewClient(cfg.LunarCrushAPIKey)
	}

	return &Provider{
		coinglass:          cg,
		whalealert:         wa,
		lunarcrush:         lc,
		symbols:            cfg.Symbols,
		signalHandlers:     make([]func(*entity.MarketSignal), 0),
		recentWhaleAlerts:  make(map[string][]*entity.WhaleAlert),
		recentLiquidations: make(map[string][]*entity.Liquidation),
		recentSentiment:    make(map[string]*entity.SocialSentiment),
	}
}

// Start starts all data source connections
func (p *Provider) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.mu.Unlock()

	// Connect CoinGlass
	if p.coinglass != nil {
		if err := p.coinglass.Connect(ctx); err != nil {
			// Log warning but continue
		}
	}

	// Connect Whale Alert
	if p.whalealert != nil {
		if err := p.whalealert.Connect(ctx); err != nil {
			// Log warning but continue
		}
	}

	// Connect LunarCrush
	if p.lunarcrush != nil {
		if err := p.lunarcrush.Connect(ctx); err != nil {
			// Log warning but continue
		}
	}

	// Start background data collection
	go p.collectData(ctx)

	// Subscribe to liquidations for each symbol
	if p.coinglass != nil {
		for _, symbol := range p.symbols {
			sym := symbol // Capture for closure
			p.coinglass.SubscribeLiquidations(ctx, symbol, func(liq *entity.Liquidation) {
				p.onLiquidation(sym, liq)
			})
		}
	}

	// Subscribe to whale alerts
	if p.whalealert != nil {
		p.whalealert.SubscribeWhaleAlerts(ctx, p.onWhaleAlert)
	}

	// Subscribe to sentiment updates
	if p.lunarcrush != nil {
		for _, symbol := range p.symbols {
			sym := symbol // Capture for closure
			p.lunarcrush.SubscribeSentiment(ctx, symbol, func(sentiment *entity.SocialSentiment) {
				p.onSentimentUpdate(sym, sentiment)
			})
		}
	}

	return nil
}

// Stop stops all data source connections
func (p *Provider) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	if p.coinglass != nil {
		p.coinglass.Disconnect(ctx)
	}
	if p.whalealert != nil {
		p.whalealert.Disconnect(ctx)
	}
	if p.lunarcrush != nil {
		p.lunarcrush.Disconnect(ctx)
	}

	return nil
}

// collectData periodically collects and broadcasts market signals
func (p *Provider) collectData(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
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

			for _, symbol := range p.symbols {
				signal, err := p.GetMarketSignal(ctx, symbol)
				if err != nil {
					continue
				}
				p.broadcastSignal(signal)
			}
		}
	}
}

// onLiquidation handles incoming liquidation events
func (p *Provider) onLiquidation(symbol string, liq *entity.Liquidation) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Keep only recent liquidations (last 10 minutes)
	cutoff := time.Now().Add(-10 * time.Minute)
	current := p.recentLiquidations[symbol]
	filtered := make([]*entity.Liquidation, 0)
	for _, l := range current {
		if l.Timestamp.After(cutoff) {
			filtered = append(filtered, l)
		}
	}
	filtered = append(filtered, liq)
	p.recentLiquidations[symbol] = filtered
}

// onWhaleAlert handles incoming whale alerts
func (p *Provider) onWhaleAlert(alert *entity.WhaleAlert) {
	p.mu.Lock()
	defer p.mu.Unlock()

	symbol := mapBlockchainToSymbol(alert.Blockchain)
	if symbol == "" {
		return
	}

	// Keep only recent alerts (last 30 minutes)
	cutoff := time.Now().Add(-30 * time.Minute)
	current := p.recentWhaleAlerts[symbol]
	filtered := make([]*entity.WhaleAlert, 0)
	for _, a := range current {
		if a.Timestamp.After(cutoff) {
			filtered = append(filtered, a)
		}
	}
	filtered = append(filtered, alert)
	p.recentWhaleAlerts[symbol] = filtered
}

// onSentimentUpdate handles incoming sentiment updates
func (p *Provider) onSentimentUpdate(symbol string, sentiment *entity.SocialSentiment) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recentSentiment[symbol] = sentiment
}

// mapBlockchainToSymbol maps blockchain name to trading symbol
func mapBlockchainToSymbol(blockchain string) string {
	switch blockchain {
	case "bitcoin":
		return "BTC"
	case "ethereum":
		return "ETH"
	case "tron":
		return "TRX"
	case "solana":
		return "SOL"
	default:
		return ""
	}
}

// GetMarketSignal returns aggregated market signal for a symbol
func (p *Provider) GetMarketSignal(ctx context.Context, symbol string) (*entity.MarketSignal, error) {
	signal := &entity.MarketSignal{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	// Get CoinGlass data
	if p.coinglass != nil {
		if oi, err := p.coinglass.GetOpenInterest(ctx, symbol); err == nil {
			signal.OpenInterest = oi
		}
		if fr, err := p.coinglass.GetFundingRate(ctx, symbol); err == nil {
			signal.FundingRate = fr
		}
		if lsr, err := p.coinglass.GetLongShortRatio(ctx, symbol); err == nil {
			signal.LongShortRatio = lsr
		}
	}

	// Get LunarCrush sentiment data
	if p.lunarcrush != nil {
		if sentiment, err := p.lunarcrush.GetSentiment(ctx, symbol); err == nil {
			signal.SocialSentiment = sentiment
		}
	}

	// Get cached whale alerts, liquidations, and sentiment
	p.mu.RLock()
	signal.RecentWhaleAlerts = p.recentWhaleAlerts[symbol]
	signal.RecentLiquidations = p.recentLiquidations[symbol]
	// Use cached sentiment if fresh API call failed
	if signal.SocialSentiment == nil {
		signal.SocialSentiment = p.recentSentiment[symbol]
	}
	p.mu.RUnlock()

	// Analyze and set bias/strength/confidence
	signal.AnalyzeSignal()

	return signal, nil
}

// SubscribeSignals subscribes to aggregated market signals
func (p *Provider) SubscribeSignals(ctx context.Context, handler func(*entity.MarketSignal)) error {
	p.mu.Lock()
	p.signalHandlers = append(p.signalHandlers, handler)
	p.mu.Unlock()
	return nil
}

// broadcastSignal broadcasts signal to all subscribers
func (p *Provider) broadcastSignal(signal *entity.MarketSignal) {
	p.mu.RLock()
	handlers := make([]func(*entity.MarketSignal), len(p.signalHandlers))
	copy(handlers, p.signalHandlers)
	p.mu.RUnlock()

	for _, handler := range handlers {
		handler(signal)
	}
}

// GetSignalSummary returns a human-readable summary of the current signal
func GetSignalSummary(signal *entity.MarketSignal) string {
	if signal == nil {
		return "No signal available"
	}

	summary := signal.Symbol + " Signal: " + string(signal.Bias)
	summary += " (Strength: " + formatPercent(signal.Strength) + ", Confidence: " + formatPercent(signal.Confidence) + ")"

	if signal.FundingRate != nil {
		summary += "\n  Funding Rate: " + formatPercent(signal.FundingRate.Rate)
	}
	if signal.LongShortRatio != nil {
		summary += "\n  Long/Short Ratio: " + formatFloat(signal.LongShortRatio.LongShortRatio)
	}
	if len(signal.RecentWhaleAlerts) > 0 {
		var inflow, outflow float64
		for _, a := range signal.RecentWhaleAlerts {
			switch a.GetAlertType() {
			case entity.WhaleAlertExchangeInflow:
				inflow += a.AmountUSD
			case entity.WhaleAlertExchangeOutflow:
				outflow += a.AmountUSD
			}
		}
		summary += "\n  Whale Inflow: $" + formatLargeNumber(inflow) + ", Outflow: $" + formatLargeNumber(outflow)
	}
	if signal.SocialSentiment != nil {
		s := signal.SocialSentiment
		sentimentStr := "neutral"
		if s.SentimentScore > 0.2 {
			sentimentStr = "bullish"
		} else if s.SentimentScore < -0.2 {
			sentimentStr = "bearish"
		}
		summary += "\n  Social Sentiment: " + sentimentStr + " (score: " + formatFloat(s.SentimentScore) + ")"
		summary += "\n  Social Volume: " + formatLargeNumber(float64(s.SocialVolume)) + " posts, " + formatLargeNumber(float64(s.Interactions)) + " interactions"
	}

	return summary
}

func formatPercent(v float64) string {
	return formatFloat(v*100) + "%"
}

func formatFloat(v float64) string {
	return string(rune(int(v*100))) + "." + string(rune(int(v*10000)%100))
}

func formatLargeNumber(v float64) string {
	if v >= 1000000000 {
		return formatFloat(v/1000000000) + "B"
	}
	if v >= 1000000 {
		return formatFloat(v/1000000) + "M"
	}
	if v >= 1000 {
		return formatFloat(v/1000) + "K"
	}
	return formatFloat(v)
}
