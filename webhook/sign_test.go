package webhook

import (
	"testing"
)

func TestSignPayload(t *testing.T) {
	sig := SignPayload("mysecret", []byte(`{"event":"test"}`))
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}
	// Deterministic: same input -> same output.
	sig2 := SignPayload("mysecret", []byte(`{"event":"test"}`))
	if sig != sig2 {
		t.Error("expected deterministic signature")
	}
}

func TestSignPayload_DifferentSecret(t *testing.T) {
	sig1 := SignPayload("secret1", []byte("data"))
	sig2 := SignPayload("secret2", []byte("data"))
	if sig1 == sig2 {
		t.Error("different secrets should produce different signatures")
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	payload := []byte(`{"call_id":"abc"}`)
	secret := "test-secret-key"
	sig := SignPayload(secret, payload)

	if !VerifySignature(secret, payload, sig) {
		t.Error("expected signature to verify")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	payload := []byte(`{"call_id":"abc"}`)
	if VerifySignature("secret", payload, "invalid-signature") {
		t.Error("expected invalid signature to fail")
	}
}

func TestSignPayload_EmptyPayload(t *testing.T) {
	sig := SignPayload("secret", []byte{})
	if sig == "" {
		t.Fatal("expected non-empty signature even for empty payload")
	}
}

func TestSignPayload_EmptySecret(t *testing.T) {
	sig := SignPayload("", []byte("data"))
	if sig == "" {
		t.Fatal("expected non-empty signature even for empty secret")
	}
}
