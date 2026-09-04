package tool

import (
	"testing"
	"time"
)

func TestMallConfirmationSignatureAndTamperDetection(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")
	want := mallConfirmation{UserID: 7, SessionID: "s", IssuedRequestID: "r1", CheckoutID: "checkout", ProductID: 9, Quantity: 2, ExpiresAtUnix: time.Now().Add(time.Minute).Unix()}
	token, err := signMallConfirmation(want, secret)
	if err != nil {
		t.Fatal(err)
	}
	got, err := verifyMallConfirmation(token, secret)
	if err != nil || got != want {
		t.Fatalf("round trip failed: got=%+v err=%v", got, err)
	}
	if _, err = verifyMallConfirmation(token+"x", secret); err == nil {
		t.Fatal("tampered token must be rejected")
	}
}

func TestMallConfirmationExpiry(t *testing.T) {
	secret := []byte("12345678901234567890123456789012")
	token, err := signMallConfirmation(mallConfirmation{UserID: 7, SessionID: "s", IssuedRequestID: "r1", CheckoutID: "checkout", ProductID: 9, Quantity: 1, ExpiresAtUnix: time.Now().Add(-time.Second).Unix()}, secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = verifyMallConfirmation(token, secret); err == nil {
		t.Fatal("expired token must be rejected")
	}
}
