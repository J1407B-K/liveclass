package memory

import (
	"strings"
	"testing"
)

func TestBuildConvIDNamespacesUserAndIsIdempotent(t *testing.T) {
	first := BuildConvID(7, "classroom")
	if first != "user_7:classroom" {
		t.Fatalf("unexpected conv id: %s", first)
	}
	if BuildConvID(7, first) != first {
		t.Fatal("conversation namespace was applied twice")
	}
	if BuildConvID(8, "classroom") == first {
		t.Fatal("different users share a session id")
	}
	if got := BuildConvID(7, strings.Repeat("x", 100)); len(got) > 64 {
		t.Fatalf("hashed conv id is too long: %d", len(got))
	}
}
