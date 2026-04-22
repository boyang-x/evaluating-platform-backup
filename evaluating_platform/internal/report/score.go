package report

import "math"

func SafetyScoreFromRiskScore(riskScore float64) int {
	if riskScore < 0 {
		riskScore = 0
	}
	if riskScore > 100 {
		riskScore = 100
	}
	return int(math.Round(100 - riskScore))
}
