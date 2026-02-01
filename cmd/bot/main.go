package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/domain/service"
	"github.com/zono819/hyperliquid-bot/internal/infrastructure/config"
	"github.com/zono819/hyperliquid-bot/internal/infrastructure/hyperliquid"
	"github.com/zono819/hyperliquid-bot/internal/infrastructure/logger"
	"github.com/zono819/hyperliquid-bot/internal/usecase/risk"
	"github.com/zono819/hyperliquid-bot/internal/usecase/strategy"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version")
	dryRun := flag.Bool("dry-run", true, "run in dry-run mode (no real orders)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("hyperliquid-bot %s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Initialize logger
	log := logger.New(logger.LevelInfo, os.Stdout)
	logger.SetDefault(log)

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Override dry-run from flag
	if *dryRun {
		log.Info("Running in DRY-RUN mode - no real orders will be placed")
	} else {
		log.Warn("Running in LIVE mode - real orders will be placed!")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Info("Received signal: %v, initiating graceful shutdown...", sig)
		cancel()
	}()

	// Run bot
	if err := run(ctx, cfg, *dryRun, log); err != nil {
		log.Error("Bot error: %v", err)
		os.Exit(1)
	}
}

// Bot represents the trading bot
type Bot struct {
	config   *config.Config
	dryRun   bool
	log      *logger.Logger

	exchange *hyperliquid.HyperliquidExchange
	strategy service.Strategy
	risk     *risk.Checker

	mu       sync.RWMutex
	running  bool
	position *entity.Position
	orders   []*entity.Order
}

func run(ctx context.Context, cfg *config.Config, dryRun bool, log *logger.Logger) error {
	log.Info("Starting %s in %s mode", cfg.App.Name, cfg.App.Environment)
	log.Info("Strategy: %s, Symbol: %s", cfg.Strategy.Name, cfg.Strategy.Symbol)

	// Create bot
	bot, err := newBot(cfg, dryRun, log)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Start bot
	if err := bot.Start(ctx); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	log.Info("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := bot.Stop(shutdownCtx); err != nil {
		log.Error("Shutdown error: %v", err)
	}

	log.Info("Bot stopped")
	return nil
}

func newBot(cfg *config.Config, dryRun bool, log *logger.Logger) (*Bot, error) {
	// Create exchange gateway
	exchangeCfg := &hyperliquid.ExchangeConfig{
		BaseURL:   cfg.Exchange.BaseURL,
		WSURL:     cfg.Exchange.WSURL,
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		Testnet:   cfg.Exchange.Testnet,
	}
	exchange := hyperliquid.NewHyperliquidExchange(exchangeCfg, log)

	// Create strategy
	strat := strategy.NewMeanReversionStrategy()

	// Create risk checker
	riskCfg := &risk.Config{
		MaxPositionSize:    cfg.Risk.MaxPositionSize,
		MaxDailyLoss:       cfg.Risk.MaxDrawdown,
		MaxConsecutiveLoss: 3,
		CooldownDuration:   5 * time.Minute,
	}
	riskChecker := risk.NewChecker(riskCfg)

	return &Bot{
		config:   cfg,
		dryRun:   dryRun,
		log:      log,
		exchange: exchange,
		strategy: strat,
		risk:     riskChecker,
	}, nil
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return fmt.Errorf("bot already running")
	}
	b.running = true
	b.mu.Unlock()

	// Initialize strategy
	if err := b.strategy.Init(ctx, b.config.Strategy.Params); err != nil {
		return fmt.Errorf("failed to init strategy: %w", err)
	}

	// Connect to exchange
	if err := b.exchange.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect exchange: %w", err)
	}

	// Subscribe to market data
	symbol := b.config.Strategy.Symbol
	if err := b.exchange.SubscribeTicker(ctx, symbol, b.onTicker); err != nil {
		return fmt.Errorf("failed to subscribe ticker: %w", err)
	}

	b.log.Info("Bot started, subscribed to %s", symbol)
	return nil
}

// Stop stops the bot
func (b *Bot) Stop(ctx context.Context) error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	b.running = false
	b.mu.Unlock()

	// Stop strategy
	if err := b.strategy.Stop(ctx); err != nil {
		b.log.Error("Failed to stop strategy: %v", err)
	}

	// Cancel all orders if not in dry-run
	if !b.dryRun {
		if err := b.exchange.CancelAllOrders(ctx, b.config.Strategy.Symbol); err != nil {
			b.log.Error("Failed to cancel orders: %v", err)
		}
	}

	// Disconnect from exchange
	if err := b.exchange.Disconnect(ctx); err != nil {
		b.log.Error("Failed to disconnect: %v", err)
	}

	return nil
}

// onTicker handles incoming ticker data - the main pipeline
func (b *Bot) onTicker(ticker *entity.Ticker) {
	b.mu.RLock()
	if !b.running {
		b.mu.RUnlock()
		return
	}
	position := b.position
	orders := b.orders
	b.mu.RUnlock()

	ctx := context.Background()

	// === PIPELINE STEP 1: Market Data → Strategy ===
	state := &service.MarketState{
		Ticker:   ticker,
		Position: position,
		Orders:   orders,
	}

	signals, err := b.strategy.OnTick(ctx, state)
	if err != nil {
		b.log.Error("Strategy error: %v", err)
		return
	}

	if len(signals) == 0 {
		return
	}

	// === PIPELINE STEP 2: Strategy Signal → Risk Check ===
	for _, sig := range signals {
		b.processSignal(ctx, sig)
	}
}

// processSignal processes a trading signal through risk check and execution
func (b *Bot) processSignal(ctx context.Context, sig *service.Signal) {
	b.log.Info("Signal: %s %s @ %.2f x %.4f - %s",
		sig.Side, sig.Symbol, sig.Price, sig.Quantity, sig.Reason)

	// Risk check: can we trade?
	check := b.risk.CanTrade()
	if !check.Allowed {
		b.log.Warn("Risk check failed: %s", check.Reason)
		return
	}

	// Risk check: position size
	sizeCheck := b.risk.CheckPositionSize(sig.Quantity)
	if !sizeCheck.Allowed {
		b.log.Warn("Position size check failed: %s", sizeCheck.Reason)
		return
	}

	// === PIPELINE STEP 3: Risk Approved → Execute Order ===
	b.executeOrder(ctx, sig)
}

// executeOrder executes an order (or simulates in dry-run mode)
func (b *Bot) executeOrder(ctx context.Context, sig *service.Signal) {
	order := &entity.Order{
		Symbol:   sig.Symbol,
		Side:     sig.Side,
		Type:     entity.OrderTypeLimit,
		Price:    sig.Price,
		Quantity: sig.Quantity,
	}

	if b.dryRun {
		// === DRY-RUN MODE: Simulate order ===
		b.log.Info("[DRY-RUN] Would place order: %s %s @ %.2f x %.4f",
			order.Side, order.Symbol, order.Price, order.Quantity)

		// Simulate filled order notification
		order.Status = entity.OrderStatusFilled
		order.FilledQty = order.Quantity
		b.strategy.OnOrderUpdate(ctx, order)
		return
	}

	// === LIVE MODE: Place real order ===
	b.log.Info("[LIVE] Placing order: %s %s @ %.2f x %.4f",
		order.Side, order.Symbol, order.Price, order.Quantity)

	result, err := b.exchange.PlaceOrder(ctx, order)
	if err != nil {
		b.log.Error("Failed to place order: %v", err)
		b.risk.RecordTrade(-0.001) // Record as small loss for consecutive tracking
		return
	}

	b.log.Info("Order placed: ID=%s, Status=%s", result.ID, result.Status)
}

// onOrderUpdate handles order status updates
func (b *Bot) onOrderUpdate(order *entity.Order) {
	b.mu.Lock()
	// Update orders list
	found := false
	for i, o := range b.orders {
		if o.ID == order.ID {
			b.orders[i] = order
			found = true
			break
		}
	}
	if !found && order.Status == entity.OrderStatusOpen {
		b.orders = append(b.orders, order)
	}
	b.mu.Unlock()

	// Notify strategy
	ctx := context.Background()
	b.strategy.OnOrderUpdate(ctx, order)

	// Track PnL for risk management
	if order.Status == entity.OrderStatusFilled {
		// Calculate PnL if this closes a position
		b.mu.RLock()
		pos := b.position
		b.mu.RUnlock()

		if pos != nil && pos.Size > 0 {
			pnl := (order.Price - pos.EntryPrice) * order.FilledQty
			if pos.Side == entity.SideSell {
				pnl = -pnl
			}
			b.risk.RecordTrade(pnl)
			b.log.Info("Trade closed: PnL=%.4f", pnl)
		}
	}
}
