package report

import "testing"

func TestSafetyScoreFromRiskScore(t *testing.T) {
	cases := []struct {
		name     string
		risk     float64
		expected int
	}{
		{name: "zero risk", risk: 0, expected: 100},
		{name: "mid risk", risk: 42.4, expected: 58},
		{name: "full risk", risk: 100, expected: 0},
		{name: "below floor", risk: -5, expected: 100},
		{name: "above ceiling", risk: 120, expected: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SafetyScoreFromRiskScore(tc.risk); got != tc.expected {
				t.Fatalf("SafetyScoreFromRiskScore(%v) = %d, want %d", tc.risk, got, tc.expected)
			}
		})
	}
}
