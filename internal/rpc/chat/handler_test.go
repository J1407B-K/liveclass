package main

import (
	"testing"
)

func TestClientMessageIDValidation(t *testing.T) {
	for _, valid := range []string{"request-123", "01994f00_test:1", "a.b"} {
		if !clientMessageIDPattern.MatchString(valid) {
			t.Errorf("valid client_message_id rejected: %q", valid)
		}
	}
	for _, invalid := range []string{"", " has-space", "中文", string(make([]byte, 129))} {
		if clientMessageIDPattern.MatchString(invalid) {
			t.Errorf("invalid client_message_id accepted: %q", invalid)
		}
	}
}
