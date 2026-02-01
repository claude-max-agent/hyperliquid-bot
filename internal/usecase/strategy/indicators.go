package strategy

import (
	"math"
)

// RSI calculates Relative Strength Index
func RSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // neutral if not enough data
	}

	gains := 0.0
	losses := 0.0

	// Calculate initial average gain/loss
	for i := 1; i <= period; i++ {
		change := prices[len(prices)-period-1+i] - prices[len(prices)-period-1+i-1]
		if change > 0 {
			gains += change
		} else {
			losses += math.Abs(change)
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))

	return rsi
}

// BollingerBands calculates Bollinger Bands
type BollingerBands struct {
	Upper  float64
	Middle float64
	Lower  float64
}

// CalculateBollingerBands returns upper, middle, lower bands
func CalculateBollingerBands(prices []float64, period int, stdDevMultiplier float64) BollingerBands {
	if len(prices) < period {
		lastPrice := prices[len(prices)-1]
		return BollingerBands{
			Upper:  lastPrice,
			Middle: lastPrice,
			Lower:  lastPrice,
		}
	}

	// Calculate SMA
	sum := 0.0
	recentPrices := prices[len(prices)-period:]
	for _, p := range recentPrices {
		sum += p
	}
	sma := sum / float64(period)

	// Calculate Standard Deviation
	variance := 0.0
	for _, p := range recentPrices {
		variance += math.Pow(p-sma, 2)
	}
	stdDev := math.Sqrt(variance / float64(period))

	return BollingerBands{
		Upper:  sma + (stdDevMultiplier * stdDev),
		Middle: sma,
		Lower:  sma - (stdDevMultiplier * stdDev),
	}
}

// SMA calculates Simple Moving Average
func SMA(prices []float64, period int) float64 {
	if len(prices) < period {
		if len(prices) == 0 {
			return 0
		}
		period = len(prices)
	}

	sum := 0.0
	recentPrices := prices[len(prices)-period:]
	for _, p := range recentPrices {
		sum += p
	}
	return sum / float64(period)
}

// EMA calculates Exponential Moving Average
func EMA(prices []float64, period int) float64 {
	if len(prices) == 0 {
		return 0
	}
	if len(prices) < period {
		return SMA(prices, len(prices))
	}

	multiplier := 2.0 / float64(period+1)
	ema := SMA(prices[:period], period)

	for i := period; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}

	return ema
}

// ATR calculates Average True Range
func ATR(highs, lows, closes []float64, period int) float64 {
	if len(highs) < 2 || len(lows) < 2 || len(closes) < 2 {
		return 0
	}

	trSum := 0.0
	count := 0
	start := len(highs) - period
	if start < 1 {
		start = 1
	}

	for i := start; i < len(highs); i++ {
		tr := math.Max(
			highs[i]-lows[i],
			math.Max(
				math.Abs(highs[i]-closes[i-1]),
				math.Abs(lows[i]-closes[i-1]),
			),
		)
		trSum += tr
		count++
	}

	if count == 0 {
		return 0
	}
	return trSum / float64(count)
}
