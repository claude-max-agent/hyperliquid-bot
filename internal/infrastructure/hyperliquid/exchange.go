package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zono819/hyperliquid-bot/internal/adapter/gateway"
	"github.com/zono819/hyperliquid-bot/internal/domain/entity"
	"github.com/zono819/hyperliquid-bot/internal/infrastructure/logger"
)

// Ensure HyperliquidExchange implements ExchangeGateway
var _ gateway.ExchangeGateway = (*HyperliquidExchange)(nil)

// ExchangeConfig contains Hyperliquid exchange configuration
type ExchangeConfig struct {
	BaseURL   string
	WSURL     string
	APIKey    string
	APISecret string
	Testnet   bool
}

// HyperliquidExchange implements ExchangeGateway for Hyperliquid
type HyperliquidExchange struct {
	config *ExchangeConfig
	client *Client
	log    *logger.Logger

	// WebSocket
	wsConn     *websocket.Conn
	wsMu       sync.RWMutex
	wsConnected bool
	wsDone     chan struct{}

	// Handlers
	tickerHandlers    map[string][]func(*entity.Ticker)
	orderbookHandlers map[string][]func(*entity.OrderBook)
	orderHandlers     []func(*entity.Order)
	handlerMu         sync.RWMutex
}

// NewHyperliquidExchange creates a new Hyperliquid exchange gateway
func NewHyperliquidExchange(config *ExchangeConfig, log *logger.Logger) *HyperliquidExchange {
	if log == nil {
		log = logger.Default()
	}

	client := NewClient(ClientConfig{
		BaseURL:   config.BaseURL,
		APIKey:    config.APIKey,
		APISecret: config.APISecret,
		Testnet:   config.Testnet,
	})

	return &HyperliquidExchange{
		config:            config,
		client:            client,
		log:               log.WithField("component", "hyperliquid"),
		tickerHandlers:    make(map[string][]func(*entity.Ticker)),
		orderbookHandlers: make(map[string][]func(*entity.OrderBook)),
	}
}

// Connect establishes connection to Hyperliquid
func (e *HyperliquidExchange) Connect(ctx context.Context) error {
	e.log.Info("Connecting to Hyperliquid (testnet: %v)", e.config.Testnet)

	// Connect WebSocket
	wsURL := e.config.WSURL
	if wsURL == "" {
		if e.config.Testnet {
			wsURL = "wss://api.hyperliquid-testnet.xyz/ws"
		} else {
			wsURL = "wss://api.hyperliquid.xyz/ws"
		}
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	e.wsMu.Lock()
	e.wsConn = conn
	e.wsConnected = true
	e.wsDone = make(chan struct{})
	e.wsMu.Unlock()

	// Start read loop
	go e.wsReadLoop()

	e.log.Info("Connected to Hyperliquid")
	return nil
}

// Disconnect closes connection to Hyperliquid
func (e *HyperliquidExchange) Disconnect(ctx context.Context) error {
	e.log.Info("Disconnecting from Hyperliquid")

	e.wsMu.Lock()
	defer e.wsMu.Unlock()

	if e.wsConn != nil {
		e.wsConnected = false
		close(e.wsDone)
		e.wsConn.Close()
		e.wsConn = nil
	}

	return nil
}

// PlaceOrder places a new order
func (e *HyperliquidExchange) PlaceOrder(ctx context.Context, order *entity.Order) (*entity.Order, error) {
	e.log.Info("Placing order: %s %s %s @ %f x %f",
		order.Symbol, order.Side, order.Type, order.Price, order.Quantity)

	// TODO: Implement order placement via REST API
	return nil, fmt.Errorf("order placement not implemented")
}

// CancelOrder cancels an order
func (e *HyperliquidExchange) CancelOrder(ctx context.Context, orderID string) error {
	e.log.Info("Canceling order: %s", orderID)
	// TODO: Implement
	return nil
}

// CancelAllOrders cancels all orders for a symbol
func (e *HyperliquidExchange) CancelAllOrders(ctx context.Context, symbol string) error {
	e.log.Info("Canceling all orders for: %s", symbol)
	// TODO: Implement
	return nil
}

// GetOrder retrieves order by ID
func (e *HyperliquidExchange) GetOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetOpenOrders retrieves all open orders
func (e *HyperliquidExchange) GetOpenOrders(ctx context.Context, symbol string) ([]*entity.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetPosition retrieves current position
func (e *HyperliquidExchange) GetPosition(ctx context.Context, symbol string) (*entity.Position, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetTicker retrieves current ticker
func (e *HyperliquidExchange) GetTicker(ctx context.Context, symbol string) (*entity.Ticker, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetOrderBook retrieves order book
func (e *HyperliquidExchange) GetOrderBook(ctx context.Context, symbol string, depth int) (*entity.OrderBook, error) {
	return nil, fmt.Errorf("not implemented")
}

// SubscribeTicker subscribes to ticker updates
func (e *HyperliquidExchange) SubscribeTicker(ctx context.Context, symbol string, handler func(*entity.Ticker)) error {
	e.handlerMu.Lock()
	e.tickerHandlers[symbol] = append(e.tickerHandlers[symbol], handler)
	e.handlerMu.Unlock()

	// Send subscription message
	msg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "allMids",
		},
	}

	return e.wsSend(msg)
}

// SubscribeOrderBook subscribes to order book updates
func (e *HyperliquidExchange) SubscribeOrderBook(ctx context.Context, symbol string, handler func(*entity.OrderBook)) error {
	e.handlerMu.Lock()
	e.orderbookHandlers[symbol] = append(e.orderbookHandlers[symbol], handler)
	e.handlerMu.Unlock()

	msg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "l2Book",
			"coin": symbol,
		},
	}

	return e.wsSend(msg)
}

// SubscribeOrders subscribes to order updates
func (e *HyperliquidExchange) SubscribeOrders(ctx context.Context, handler func(*entity.Order)) error {
	e.handlerMu.Lock()
	e.orderHandlers = append(e.orderHandlers, handler)
	e.handlerMu.Unlock()

	// TODO: Implement user order subscription
	return nil
}

// wsSend sends a message via WebSocket
func (e *HyperliquidExchange) wsSend(msg interface{}) error {
	e.wsMu.RLock()
	conn := e.wsConn
	connected := e.wsConnected
	e.wsMu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	e.wsMu.Lock()
	defer e.wsMu.Unlock()
	return e.wsConn.WriteMessage(websocket.TextMessage, data)
}

// wsReadLoop reads messages from WebSocket
func (e *HyperliquidExchange) wsReadLoop() {
	for {
		e.wsMu.RLock()
		conn := e.wsConn
		done := e.wsDone
		e.wsMu.RUnlock()

		if conn == nil {
			return
		}

		select {
		case <-done:
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				e.log.Error("WebSocket read error: %v", err)
			}
			return
		}

		e.handleWSMessage(message)
	}
}

// handleWSMessage processes incoming WebSocket messages
func (e *HyperliquidExchange) handleWSMessage(data []byte) {
	var msg struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	switch msg.Channel {
	case "allMids":
		e.handleAllMids(msg.Data)
	case "l2Book":
		e.handleL2Book(msg.Data)
	}
}

// handleAllMids processes ticker data
func (e *HyperliquidExchange) handleAllMids(data json.RawMessage) {
	var midsData struct {
		Mids map[string]string `json:"mids"`
	}
	if err := json.Unmarshal(data, &midsData); err != nil {
		return
	}

	e.handlerMu.RLock()
	defer e.handlerMu.RUnlock()

	for symbol, handlers := range e.tickerHandlers {
		midStr, ok := midsData.Mids[symbol]
		if !ok {
			continue
		}

		var mid float64
		fmt.Sscanf(midStr, "%f", &mid)

		ticker := &entity.Ticker{
			Symbol:    symbol,
			LastPrice: mid,
			BidPrice:  mid,
			AskPrice:  mid,
			Timestamp: time.Now(),
		}

		for _, h := range handlers {
			h(ticker)
		}
	}
}

// handleL2Book processes order book data
func (e *HyperliquidExchange) handleL2Book(data json.RawMessage) {
	var bookData struct {
		Coin   string `json:"coin"`
		Levels [][]struct {
			Px string `json:"px"`
			Sz string `json:"sz"`
		} `json:"levels"`
		Time int64 `json:"time"`
	}
	if err := json.Unmarshal(data, &bookData); err != nil {
		return
	}

	e.handlerMu.RLock()
	handlers := e.orderbookHandlers[bookData.Coin]
	e.handlerMu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	ob := &entity.OrderBook{
		Symbol:    bookData.Coin,
		Timestamp: time.UnixMilli(bookData.Time),
		Bids:      make([]entity.OrderBookLevel, 0),
		Asks:      make([]entity.OrderBookLevel, 0),
	}

	if len(bookData.Levels) >= 2 {
		for _, lvl := range bookData.Levels[0] {
			var px, sz float64
			fmt.Sscanf(lvl.Px, "%f", &px)
			fmt.Sscanf(lvl.Sz, "%f", &sz)
			ob.Bids = append(ob.Bids, entity.OrderBookLevel{Price: px, Size: sz})
		}
		for _, lvl := range bookData.Levels[1] {
			var px, sz float64
			fmt.Sscanf(lvl.Px, "%f", &px)
			fmt.Sscanf(lvl.Sz, "%f", &sz)
			ob.Asks = append(ob.Asks, entity.OrderBookLevel{Price: px, Size: sz})
		}
	}

	for _, h := range handlers {
		h(ob)
	}
}
