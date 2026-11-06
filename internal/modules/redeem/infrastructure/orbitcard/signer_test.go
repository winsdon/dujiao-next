package orbitcard

import "testing"

func TestSignCanonicalStable(t *testing.T) {
	sig := SignCanonical(
		"secret",
		"key",
		"1700000000",
		"nonce1234567890ab",
		"POST",
		"orbitcard.cc",
		"/api/open/v1/getAccountBalance",
		"",
		BodySHA256Hex([]byte("{}")),
		"",
	)
	if len(sig) != 64 {
		t.Fatalf("expected 64 hex chars, got %d (%s)", len(sig), sig)
	}
	sig2 := SignCanonical(
		"secret", "key", "1700000000", "nonce1234567890ab",
		"POST", "orbitcard.cc", "/api/open/v1/getAccountBalance", "",
		BodySHA256Hex([]byte("{}")), "",
	)
	if sig != sig2 {
		t.Fatalf("signature not stable")
	}
}
