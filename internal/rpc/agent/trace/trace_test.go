package trace

import (
	"errors"
	"strings"
	"testing"
)

func TestSafeErrorRedactsProviderLimitPayload(t *testing.T) {
	got := SafeError(errors.New(`remote SetLimitExceeded account=123 request_id=secret`))
	if got != "provider inference limit exceeded" {
		t.Fatalf("SafeError()=%q", got)
	}
	long := SafeError(errors.New(strings.Repeat("x", 600)))
	if len(long) > 515 || !strings.HasSuffix(long, "...") {
		t.Fatalf("long error was not bounded: %d", len(long))
	}
}
