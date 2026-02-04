package entity

import "time"

// FOMCMeeting represents an FOMC meeting with rate probabilities
type FOMCMeeting struct {
	MeetingDate     time.Time              `json:"meeting_date"`
	CurrentRate     float64                `json:"current_rate"`
	Probabilities   map[float64]float64    `json:"probabilities"` // rate -> probability
	MostLikelyRate  float64                `json:"most_likely_rate"`
	MostLikelyProb  float64                `json:"most_likely_prob"`
	RateChangeProb  float64                `json:"rate_change_prob"` // Probability of any change
	HikeProb        float64                `json:"hike_prob"`        // Probability of rate hike
	CutProb         float64                `json:"cut_prob"`         // Probability of rate cut
	HoldProb        float64                `json:"hold_prob"`        // Probability of no change
	Timestamp       time.Time              `json:"timestamp"`
}

// FedWatchData represents aggregated FedWatch data
type FedWatchData struct {
	CurrentRate      float64        `json:"current_rate"`
	NextMeeting      *FOMCMeeting   `json:"next_meeting"`
	UpcomingMeetings []*FOMCMeeting `json:"upcoming_meetings"`
	Timestamp        time.Time      `json:"timestamp"`
}

// EconomicIndicator represents an economic indicator value
type EconomicIndicator struct {
	Country       string    `json:"country"`
	Category      string    `json:"category"`      // e.g., "CPI", "GDP", "Unemployment"
	Name          string    `json:"name"`
	Value         float64   `json:"value"`
	Previous      float64   `json:"previous"`
	Forecast      float64   `json:"forecast"`
	Unit          string    `json:"unit"`
	Frequency     string    `json:"frequency"`     // e.g., "Monthly", "Quarterly"
	LastUpdate    time.Time `json:"last_update"`
	NextRelease   time.Time `json:"next_release"`
	Importance    string    `json:"importance"`    // "high", "medium", "low"
	Timestamp     time.Time `json:"timestamp"`
}

// EconomicEvent represents a scheduled economic event/release
type EconomicEvent struct {
	ID          string    `json:"id"`
	Country     string    `json:"country"`
	Category    string    `json:"category"`
	Event       string    `json:"event"`
	Date        time.Time `json:"date"`
	Actual      *float64  `json:"actual,omitempty"`
	Previous    float64   `json:"previous"`
	Forecast    float64   `json:"forecast"`
	Importance  string    `json:"importance"` // "high", "medium", "low"
	Impact      string    `json:"impact"`     // "positive", "negative", "neutral"
}

// MacroSignal represents aggregated macro signal for trading
type MacroSignal struct {
	Timestamp time.Time `json:"timestamp"`

	// FedWatch data
	FedWatch *FedWatchData `json:"fed_watch,omitempty"`

	// Key economic indicators
	CPI          *EconomicIndicator `json:"cpi,omitempty"`
	GDP          *EconomicIndicator `json:"gdp,omitempty"`
	Unemployment *EconomicIndicator `json:"unemployment,omitempty"`
	PCE          *EconomicIndicator `json:"pce,omitempty"` // Fed's preferred inflation measure

	// Upcoming events
	UpcomingEvents []*EconomicEvent `json:"upcoming_events,omitempty"`

	// Aggregated signal
	Bias       SignalBias `json:"bias"`
	Strength   float64    `json:"strength"`
	Confidence float64    `json:"confidence"`
}

// AnalyzeMacroSignal analyzes macro data and sets bias/strength
func (m *MacroSignal) AnalyzeMacroSignal() {
	var bullishScore, bearishScore float64
	var dataPoints int

	// Analyze FedWatch data
	if m.FedWatch != nil && m.FedWatch.NextMeeting != nil {
		dataPoints++
		meeting := m.FedWatch.NextMeeting

		// Rate cuts are generally bullish for risk assets
		if meeting.CutProb > 0.5 {
			bullishScore += 0.3 * meeting.CutProb
		}
		// Rate hikes are bearish
		if meeting.HikeProb > 0.3 {
			bearishScore += 0.3 * meeting.HikeProb
		}
	}

	// Analyze CPI (inflation)
	if m.CPI != nil {
		dataPoints++
		// Higher than expected inflation = bearish (more rate hikes expected)
		if m.CPI.Value > m.CPI.Forecast && m.CPI.Forecast > 0 {
			bearishScore += 0.2
		}
		// Lower than expected = bullish
		if m.CPI.Value < m.CPI.Forecast && m.CPI.Forecast > 0 {
			bullishScore += 0.2
		}
	}

	// Analyze GDP
	if m.GDP != nil {
		dataPoints++
		// Strong GDP = bullish
		if m.GDP.Value > m.GDP.Previous {
			bullishScore += 0.15
		}
		// Weak GDP = bearish
		if m.GDP.Value < m.GDP.Previous {
			bearishScore += 0.15
		}
	}

	// Analyze Unemployment
	if m.Unemployment != nil {
		dataPoints++
		// Rising unemployment = bearish for economy but could be bullish for rates
		if m.Unemployment.Value > m.Unemployment.Previous {
			// Mixed signal - weak economy but potential rate cuts
			bullishScore += 0.1  // Rate cut expectations
			bearishScore += 0.1 // Economic weakness
		}
		// Falling unemployment = strong economy
		if m.Unemployment.Value < m.Unemployment.Previous {
			bullishScore += 0.1
		}
	}

	// Calculate final signal
	totalScore := bullishScore + bearishScore
	if totalScore == 0 || dataPoints == 0 {
		m.Bias = SignalBiasNeutral
		m.Strength = 0
		m.Confidence = 0
		return
	}

	if bullishScore > bearishScore {
		m.Bias = SignalBiasBullish
		m.Strength = (bullishScore - bearishScore) / totalScore
	} else if bearishScore > bullishScore {
		m.Bias = SignalBiasBearish
		m.Strength = (bearishScore - bullishScore) / totalScore
	} else {
		m.Bias = SignalBiasNeutral
		m.Strength = 0
	}

	// Confidence based on data availability (4 possible data points)
	m.Confidence = float64(dataPoints) / 4.0
}

// GetFedBias returns the market bias based on Fed policy expectations
func GetFedBias(fedWatch *FedWatchData) (SignalBias, float64) {
	if fedWatch == nil || fedWatch.NextMeeting == nil {
		return SignalBiasNeutral, 0
	}

	meeting := fedWatch.NextMeeting

	// Strong rate cut expectations = bullish for crypto
	if meeting.CutProb > 0.7 {
		return SignalBiasBullish, meeting.CutProb
	}
	if meeting.CutProb > 0.5 {
		return SignalBiasBullish, meeting.CutProb * 0.7
	}

	// Strong rate hike expectations = bearish for crypto
	if meeting.HikeProb > 0.5 {
		return SignalBiasBearish, meeting.HikeProb
	}
	if meeting.HikeProb > 0.3 {
		return SignalBiasBearish, meeting.HikeProb * 0.7
	}

	return SignalBiasNeutral, 0
}
