package eval

import "testing"

func TestShouldJudgeAlwaysSelectsAnomaly(t *testing.T) {
	selected, reasons := ShouldJudge(OnlinePolicy{MaxTokens: 100, MaxRetries: 1}, Observation{RequestID: "r", Tokens: 101})
	if !selected || len(reasons) != 1 || reasons[0] != "token_spike" {
		t.Fatalf("selected=%v reasons=%v", selected, reasons)
	}
}
func TestShouldJudgeCanDisableRandomSampling(t *testing.T) {
	selected, _ := ShouldJudge(OnlinePolicy{MaxTokens: 100}, Observation{RequestID: "r", Tokens: 10})
	if selected {
		t.Fatal("unexpected sample")
	}
}
