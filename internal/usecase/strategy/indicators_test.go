package strategy

import (
	"math"
	"testing"
)

func TestRSI(t *testing.T) {
	tests := []struct {
		name     string
		prices   []float64
		period   int
		expected float64
		delta    float64
	}{
		{
			name:     "all gains",
			prices:   []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115},
			period:   14,
			expected: 100.0,
			delta:    0.1,
		},
		{
			name:     "all losses",
			prices:   []float64{115, 114, 113, 112, 111, 110, 109, 108, 107, 106, 105, 104, 103, 102, 101, 100},
			period:   14,
			expected: 0.0,
			delta:    0.1,
		},
		{
			name:     "neutral",
			prices:   []float64{100, 101, 100, 101, 100, 101, 100, 101, 100, 101, 100, 101, 100, 101, 100},
			period:   14,
			expected: 50.0,
			delta:    1.0,
		},
		{
			name:     "insufficient data",
			prices:   []float64{100, 101, 102},
			period:   14,
			expected: 50.0, // returns neutral when not enough data
			delta:    0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RSI(tt.prices, tt.period)
			if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("RSI() = %v, expected %v (±%v)", result, tt.expected, tt.delta)
			}
		})
	}
}

func TestCalculateBollingerBands(t *testing.T) {
	tests := []struct {
		name       string
		prices     []float64
		period     int
		stdDev     float64
		wantUpper  float64
		wantMiddle float64
		wantLower  float64
		delta      float64
	}{
		{
			name:       "constant prices",
			prices:     []float64{100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
			period:     10,
			stdDev:     2.0,
			wantUpper:  100.0,
			wantMiddle: 100.0,
			wantLower:  100.0,
			delta:      0.01,
		},
		{
			name:       "increasing prices",
			prices:     []float64{90, 92, 94, 96, 98, 100, 102, 104, 106, 108},
			period:     10,
			stdDev:     2.0,
			wantMiddle: 99.0,
			delta:      0.5,
		},
		{
			name:       "insufficient data",
			prices:     []float64{100, 101, 102},
			period:     10,
			stdDev:     2.0,
			wantUpper:  102.0,
			wantMiddle: 102.0,
			wantLower:  102.0,
			delta:      0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBollingerBands(tt.prices, tt.period, tt.stdDev)

			if tt.wantMiddle > 0 && math.Abs(result.Middle-tt.wantMiddle) > tt.delta {
				t.Errorf("BB Middle = %v, expected %v (±%v)", result.Middle, tt.wantMiddle, tt.delta)
			}

			// Upper should be >= Middle >= Lower
			if result.Upper < result.Middle {
				t.Errorf("BB Upper (%v) should be >= Middle (%v)", result.Upper, result.Middle)
			}
			if result.Middle < result.Lower {
				t.Errorf("BB Middle (%v) should be >= Lower (%v)", result.Middle, result.Lower)
			}
		})
	}
}

func TestSMA(t *testing.T) {
	tests := []struct {
		name     string
		prices   []float64
		period   int
		expected float64
	}{
		{
			name:     "simple average",
			prices:   []float64{10, 20, 30, 40, 50},
			period:   5,
			expected: 30.0,
		},
		{
			name:     "partial period",
			prices:   []float64{10, 20, 30},
			period:   5,
			expected: 20.0,
		},
		{
			name:     "empty prices",
			prices:   []float64{},
			period:   5,
			expected: 0.0,
		},
		{
			name:     "uses last N prices",
			prices:   []float64{100, 10, 20, 30, 40, 50},
			period:   5,
			expected: 30.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SMA(tt.prices, tt.period)
			if result != tt.expected {
				t.Errorf("SMA() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEMA(t *testing.T) {
	tests := []struct {
		name   string
		prices []float64
		period int
	}{
		{
			name:   "basic EMA",
			prices: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			period: 5,
		},
		{
			name:   "empty prices",
			prices: []float64{},
			period: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EMA(tt.prices, tt.period)

			if len(tt.prices) == 0 {
				if result != 0 {
					t.Errorf("EMA() for empty prices = %v, expected 0", result)
				}
				return
			}

			// EMA should be between min and max prices
			minPrice := tt.prices[0]
			maxPrice := tt.prices[0]
			for _, p := range tt.prices {
				if p < minPrice {
					minPrice = p
				}
				if p > maxPrice {
					maxPrice = p
				}
			}

			if result < minPrice || result > maxPrice {
				t.Errorf("EMA() = %v, expected between %v and %v", result, minPrice, maxPrice)
			}
		})
	}
}

func TestATR(t *testing.T) {
	tests := []struct {
		name   string
		highs  []float64
		lows   []float64
		closes []float64
		period int
	}{
		{
			name:   "basic ATR",
			highs:  []float64{105, 106, 107, 108, 109, 110},
			lows:   []float64{95, 96, 97, 98, 99, 100},
			closes: []float64{100, 101, 102, 103, 104, 105},
			period: 5,
		},
		{
			name:   "insufficient data",
			highs:  []float64{105},
			lows:   []float64{95},
			closes: []float64{100},
			period: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ATR(tt.highs, tt.lows, tt.closes, tt.period)

			if len(tt.highs) < 2 {
				if result != 0 {
					t.Errorf("ATR() for insufficient data = %v, expected 0", result)
				}
				return
			}

			// ATR should be positive
			if result < 0 {
				t.Errorf("ATR() = %v, expected positive value", result)
			}
		})
	}
}
