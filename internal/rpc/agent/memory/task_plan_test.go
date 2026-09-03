package memory

import "testing"

func TestValidTaskTransition(t *testing.T) {
	for _, tc := range []struct {
		from, to string
		valid    bool
	}{{"pending", "running", true}, {"running", "done", true}, {"done", "running", false}, {"pending", "done", false}} {
		if got := validTaskTransition(tc.from, tc.to); got != tc.valid {
			t.Fatalf("%s -> %s = %v", tc.from, tc.to, got)
		}
	}
}
