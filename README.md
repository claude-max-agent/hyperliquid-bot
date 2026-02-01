# Hyperliquid HFT Bot

High-frequency trading bot for Hyperliquid exchange, built with Go and Clean Architecture.

## Architecture

```
cmd/bot/               # Entry point
internal/
├── domain/            # Business logic (exchange-agnostic)
│   ├── entity/        # Core entities (Order, Position, etc.)
│   ├── repository/    # Repository interfaces
│   └── service/       # Domain services (Strategy interface)
├── usecase/           # Application use cases
│   ├── bot.go         # Bot orchestration
│   └── strategy/      # Trading strategies
│       ├── indicators.go      # Technical indicators
│       └── mean_reversion.go  # Mean Reversion strategy
├── adapter/           # Interface adapters
│   ├── controller/    # HTTP/CLI controllers
│   ├── gateway/       # External service interfaces
│   └── presenter/     # Output formatters
└── infrastructure/    # Frameworks & drivers
    ├── exchange/      # Exchange implementations
    ├── config/        # Configuration loader
    └── logger/        # Logging
```

## Quick Start

### Prerequisites

- Go 1.22+
- Make

### Setup

```bash
# Clone repository
git clone https://github.com/zono819/hyperliquid-bot.git
cd hyperliquid-bot

# Install dependencies
make deps

# Copy and edit config
cp config/config.example.yaml config/config.yaml
cp .env.example .env
# Edit .env with your API credentials

# Build
make build

# Run
make run
```

## Trading Strategies

### Mean Reversion Strategy

Mean Reversion strategy is designed for small capital ($100) with controlled risk.

#### Why Mean Reversion?

| Criteria | Evaluation | Reason |
|----------|------------|--------|
| Win Rate | 55-65% | Higher than trend-following; better for small capital |
| Entry Signal | Clear | RSI + Bollinger Bands extremes are objective |
| Trade Frequency | 3-10/day | Balanced fee burden and opportunities |
| Hold Time | Short | Avoid funding rate costs |
| Implementation | Simple | Few indicators, easy to backtest |

#### Entry/Exit Logic

```
┌─────────────────────────────────────────────────────────────┐
│                    MARKET DATA FLOW                         │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   Update Price History │
              │   (last 100 prices)    │
              └───────────┬───────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  Calculate Indicators  │
              │  - RSI (14 period)     │
              │  - Bollinger Bands     │
              │    (20 period, 2.5σ)   │
              └───────────┬───────────┘
                          │
          ┌───────────────┴───────────────┐
          │                               │
          ▼                               ▼
   ┌─────────────┐                 ┌─────────────┐
   │ Has Position │                │ No Position  │
   │    = YES     │                │    = NO      │
   └──────┬──────┘                 └──────┬──────┘
          │                               │
          ▼                               ▼
   ┌─────────────────┐           ┌──────────────────┐
   │ Check Exit      │           │ Check Entry      │
   │ - Take Profit?  │           │                  │
   │ - Stop Loss?    │           │ RSI < 25 AND     │
   │ - Timeout?      │           │ Price < BB_Lower │
   └────────┬────────┘           │ → LONG ENTRY     │
            │                    │                  │
            ▼                    │ RSI > 75 AND     │
   ┌─────────────────┐           │ Price > BB_Upper │
   │ Generate Exit   │           │ → SHORT ENTRY    │
   │ Signal          │           └────────┬─────────┘
   └─────────────────┘                    │
                                          ▼
                               ┌─────────────────┐
                               │ Generate Entry  │
                               │ Signal          │
                               └─────────────────┘
```

#### Configuration

| Parameter | Key | Default | Description |
|-----------|-----|---------|-------------|
| RSI Period | `rsi_period` | 14 | RSI calculation period |
| RSI Oversold | `rsi_oversold` | 25.0 | Long entry threshold (< this value) |
| RSI Overbought | `rsi_overbought` | 75.0 | Short entry threshold (> this value) |
| BB Period | `bb_period` | 20 | Bollinger Bands SMA period |
| BB Std Dev | `bb_std_dev` | 2.5 | Bollinger Bands multiplier |
| Take Profit | `take_profit_pct` | 0.004 | Take profit (0.4%) |
| Stop Loss | `stop_loss_pct` | 0.0025 | Stop loss (0.25%) |
| Max Hold Time | `max_hold_time` | 1800 | Timeout in seconds (30 min) |
| Position Size | `position_size` | 0.001 | Quantity per trade |
| Max Positions | `max_positions` | 1 | Concurrent positions limit |

#### Supported Symbols

- BTC/USDC (BTC, BTC/USDC, BTC-PERP, BTCUSDC)
- ETH/USDC (ETH, ETH/USDC, ETH-PERP, ETHUSDC)
- XRP/USDC (XRP, XRP/USDC, XRP-PERP, XRPUSDC)

#### Example Configuration

```yaml
strategy:
  name: "mean_reversion"
  config:
    rsi_period: 14
    rsi_oversold: 25
    rsi_overbought: 75
    bb_period: 20
    bb_std_dev: 2.5
    take_profit_pct: 0.004
    stop_loss_pct: 0.0025
    max_hold_time: 1800
    position_size: 0.001
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `EXCHANGE_API_KEY` | Exchange API key |
| `EXCHANGE_API_SECRET` | Exchange API secret |
| `EXCHANGE_TESTNET` | Use testnet (true/false) |
| `LOG_LEVEL` | Log level (debug/info/warn/error) |
| `RISK_MAX_LEVERAGE` | Maximum leverage |

## Strategy Interface

Implement the `Strategy` interface to create custom trading strategies:

```go
type Strategy interface {
    // Name returns strategy identifier
    Name() string

    // Init initializes strategy with runtime config
    Init(ctx context.Context, config map[string]interface{}) error

    // OnTick processes market data and returns trading signals
    OnTick(ctx context.Context, state *MarketState) ([]*Signal, error)

    // OnOrderUpdate handles order status changes
    OnOrderUpdate(ctx context.Context, order *entity.Order) error

    // OnPositionUpdate handles position changes
    OnPositionUpdate(ctx context.Context, position *entity.Position) error

    // Stop performs cleanup
    Stop(ctx context.Context) error
}
```

### Creating a New Strategy

1. Create a new file in `internal/usecase/strategy/`
2. Implement the `Strategy` interface
3. Register in strategy factory

## Technical Indicators

Available indicators in `internal/usecase/strategy/indicators.go`:

| Indicator | Function | Parameters |
|-----------|----------|------------|
| RSI | `RSI(prices, period)` | prices: []float64, period: int |
| Bollinger Bands | `CalculateBollingerBands(prices, period, stdDev)` | prices: []float64, period: int, stdDev: float64 |
| SMA | `SMA(prices, period)` | prices: []float64, period: int |
| EMA | `EMA(prices, period)` | prices: []float64, period: int |
| ATR | `ATR(highs, lows, closes, period)` | OHLC data, period: int |

## Risk Management

### Kill Switch Conditions

| Condition | Threshold | Action |
|-----------|-----------|--------|
| Daily Loss | -5% | Pause trading for 24h |
| Weekly Loss | -15% | Pause trading for 7 days |
| Consecutive Losses | 5 | Pause and alert |
| Max Drawdown | -30% | Full stop |

### Position Management

- Maximum 1 concurrent position (configurable)
- Recommended leverage: 5x (start conservative)
- Per-trade risk: 2.5% of capital

## Development

```bash
# Run tests
make test

# Run specific tests
go test ./internal/usecase/strategy/... -v

# Run linter
make lint

# Format code
make fmt

# Install dev tools
make tools
```

## Testing

### Unit Tests

```bash
go test ./internal/usecase/strategy/... -v
```

### Backtest (Planned)

```bash
# Coming soon
make backtest SYMBOL=BTC START=2025-01-01 END=2025-12-31
```

## Deployment

### Paper Trading (Recommended First)

```bash
EXCHANGE_TESTNET=true make run
```

### Live Trading

```bash
# Start with minimal capital
EXCHANGE_TESTNET=false make run
```

## Fee Structure (Hyperliquid)

| Type | Rate |
|------|------|
| Taker | 0.045% |
| Maker | 0.015% |
| Gas | Free |

Break-even calculation:
- Round-trip fee: 0.045% × 2 = 0.09%
- Minimum profitable move: > 0.09%

## License

MIT
