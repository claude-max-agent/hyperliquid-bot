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

## Configuration

Configuration supports both YAML files and environment variables. Environment variables take precedence.

### YAML Configuration

See `config/config.example.yaml` for full options.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `EXCHANGE_API_KEY` | Exchange API key |
| `EXCHANGE_API_SECRET` | Exchange API secret |
| `EXCHANGE_TESTNET` | Use testnet (true/false) |
| `LOG_LEVEL` | Log level (debug/info/warn/error) |
| `RISK_MAX_LEVERAGE` | Maximum leverage |

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Install dev tools
make tools
```

## Strategy Interface

Implement the `Strategy` interface to create custom trading strategies:

```go
type Strategy interface {
    Name() string
    Init(ctx context.Context, config map[string]interface{}) error
    OnTick(ctx context.Context, state *MarketState) ([]*Signal, error)
    OnOrderUpdate(ctx context.Context, order *entity.Order) error
    OnPositionUpdate(ctx context.Context, position *entity.Position) error
    Stop(ctx context.Context) error
}
```

## License

MIT
