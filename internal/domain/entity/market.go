package entity

import (
	"time"
)

// Ticker represents market ticker data (exchange-agnostic)
type Ticker struct {
	Symbol    string
	BidPrice  float64
	BidSize   float64
	AskPrice  float64
	AskSize   float64
	LastPrice float64
	Volume24h float64
	Timestamp time.Time
}

// Spread returns bid-ask spread
func (t *Ticker) Spread() float64 {
	return t.AskPrice - t.BidPrice
}

// SpreadBps returns spread in basis points
func (t *Ticker) SpreadBps() float64 {
	if t.MidPrice() == 0 {
		return 0
	}
	return (t.Spread() / t.MidPrice()) * 10000
}

// MidPrice returns mid price
func (t *Ticker) MidPrice() float64 {
	return (t.BidPrice + t.AskPrice) / 2
}

// OrderBookLevel represents a single level in order book
type OrderBookLevel struct {
	Price float64
	Size  float64
}

// OrderBook represents order book data (exchange-agnostic)
type OrderBook struct {
	Symbol    string
	Bids      []OrderBookLevel
	Asks      []OrderBookLevel
	Timestamp time.Time
}

// BestBid returns best bid price and size
func (ob *OrderBook) BestBid() (float64, float64) {
	if len(ob.Bids) == 0 {
		return 0, 0
	}
	return ob.Bids[0].Price, ob.Bids[0].Size
}

// BestAsk returns best ask price and size
func (ob *OrderBook) BestAsk() (float64, float64) {
	if len(ob.Asks) == 0 {
		return 0, 0
	}
	return ob.Asks[0].Price, ob.Asks[0].Size
}

// Candle represents OHLCV candle data
type Candle struct {
	Symbol    string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp time.Time
}
