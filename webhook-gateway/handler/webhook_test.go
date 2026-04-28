package handler

import "testing"

func TestNormalizeTicker(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "plain spot-like symbol", input: "XRPUSDT", expect: "XRPUSDT"},
		{name: "tradingview perpetual suffix", input: "XRPUSDT.P", expect: "XRPUSDT"},
		{name: "lowercase with whitespace", input: " xrpusdt ", expect: "XRPUSDT"},
		{name: "lowercase perpetual with whitespace", input: " xrpusdt.p ", expect: "XRPUSDT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTicker(tt.input); got != tt.expect {
				t.Fatalf("normalizeTicker(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}
