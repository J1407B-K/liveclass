package dao

import (
	"testing"
	"time"
)

func TestHistoryCursorRoundTrip(t *testing.T) {
	want := historyCursor{
		CreatedAt: time.Date(2026, time.August, 31, 12, 0, 0, 123, time.UTC),
		MessageID: "01994f00-test",
	}
	encoded, err := encodeCursor(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.MessageID != want.MessageID {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDecodeCursorRejectsInvalidValues(t *testing.T) {
	for _, cursor := range []string{"not-base64!", "e30"} {
		if _, err := decodeCursor(cursor); err == nil {
			t.Fatalf("accepted invalid cursor %q", cursor)
		}
	}
}
