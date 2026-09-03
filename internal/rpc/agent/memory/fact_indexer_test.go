package memory

import (
	"liveclass/internal/rpc/agent/model"
	"testing"
)

func TestShouldSupersedeUsesConfidenceAndSource(t *testing.T) {
	old := model.UserFact{Confidence: 0.9, Source: "conversation"}
	if shouldSupersede(old, FactWrite{Confidence: 0.6, Source: "user_explicit"}) {
		t.Fatal("low-confidence fact superseded active memory")
	}
	if !shouldSupersede(old, FactWrite{Confidence: 0.8, Source: "user_explicit"}) {
		t.Fatal("higher-priority explicit fact should supersede")
	}
	if !shouldSupersede(old, FactWrite{Confidence: 0.95, Source: "conversation"}) {
		t.Fatal("higher-confidence newer fact should supersede")
	}
}

func TestAllowedFactTypeIsClosedSet(t *testing.T) {
	if !allowedFactType("preference") || !allowedFactType("episodic") || allowedFactType("任意类型") {
		t.Fatal("fact type validation is not deterministic")
	}
}
