package gateway

import (
	"context"

	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
)

// DataSourceGateway defines external data source interface
type DataSourceGateway interface {
	// Connect establishes connection to data source
	Connect(ctx context.Context) error

	// Disconnect closes connection
	Disconnect(ctx context.Context) error

	// GetLiquidations retrieves recent liquidations
	GetLiquidations(ctx context.Context, symbol string) ([]*entity.Liquidation, error)

	// GetOpenInterest retrieves open interest data
	GetOpenInterest(ctx context.Context, symbol string) (*entity.OpenInterest, error)

	// GetFundingRate retrieves funding rate data
	GetFundingRate(ctx context.Context, symbol string) (*entity.FundingRate, error)

	// GetLongShortRatio retrieves long/short ratio
	GetLongShortRatio(ctx context.Context, symbol string) (*entity.LongShortRatio, error)

	// SubscribeLiquidations subscribes to liquidation events
	SubscribeLiquidations(ctx context.Context, symbol string, handler func(*entity.Liquidation)) error

	// SubscribeWhaleAlerts subscribes to whale transaction alerts
	SubscribeWhaleAlerts(ctx context.Context, handler func(*entity.WhaleAlert)) error
}

// MarketSignalProvider aggregates multiple data sources for trading signals
type MarketSignalProvider interface {
	// Start starts all data source connections
	Start(ctx context.Context) error

	// Stop stops all data source connections
	Stop(ctx context.Context) error

	// GetMarketSignal returns aggregated market signal
	GetMarketSignal(ctx context.Context, symbol string) (*entity.MarketSignal, error)

	// SubscribeSignals subscribes to aggregated market signals
	SubscribeSignals(ctx context.Context, handler func(*entity.MarketSignal)) error
}
