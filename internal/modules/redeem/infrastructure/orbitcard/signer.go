package orbitcard

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SignCanonical builds ORBITCARD-HMAC-SHA256-V1 signature (lowercase hex).
func SignCanonical(secret, apiKey, timestamp, nonce, method, host, path, query, bodySHA256Hex, idempotencyKey string) string {
	canonical := strings.Join([]string{
		"ORBITCARD-HMAC-SHA256-V1",
		apiKey,
		timestamp,
		nonce,
		strings.ToUpper(strings.TrimSpace(method)),
		strings.TrimSpace(host),
		path,
		query,
		bodySHA256Hex,
		idempotencyKey,
	}, "\n")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

func BodySHA256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
