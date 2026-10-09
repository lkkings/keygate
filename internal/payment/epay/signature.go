package epay

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// SignatureAlgorithm defines the signature algorithm type
type SignatureAlgorithm string

const (
	SignatureMD5        SignatureAlgorithm = "MD5"
	SignatureHMACSHA256 SignatureAlgorithm = "HMAC-SHA256"
)

// Sign generates a signature using the specified algorithm
// NOTE: This implementation is based on common payment gateway patterns.
// The actual ePay signature algorithm may differ and should be updated
// once official documentation is available.
func Sign(params map[string]string, apiKey string, algorithm SignatureAlgorithm) (string, error) {
	// Remove sign field if present
	delete(params, "sign")
	delete(params, "Sign")

	// Remove empty values
	filtered := make(map[string]string)
	for k, v := range params {
		if v != "" {
			filtered[k] = v
		}
	}

	// Sort keys alphabetically
	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build string to sign: key1=value1&key2=value2&key=apiKey
	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(filtered[k])
	}

	// Append API key
	builder.WriteString("&key=")
	builder.WriteString(apiKey)

	stringToSign := builder.String()

	// Compute signature based on algorithm
	switch algorithm {
	case SignatureMD5:
		return signMD5(stringToSign), nil
	case SignatureHMACSHA256:
		return signHMACSHA256(stringToSign, apiKey), nil
	default:
		return "", fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}
}

// VerifySign verifies a signature
func VerifySign(params map[string]string, apiKey string, algorithm SignatureAlgorithm) error {
	receivedSign := params["sign"]
	if receivedSign == "" {
		return fmt.Errorf("missing sign field")
	}

	expectedSign, err := Sign(params, apiKey, algorithm)
	if err != nil {
		return fmt.Errorf("failed to compute signature: %w", err)
	}

	// Case-insensitive comparison for MD5 (might be uppercase or lowercase)
	if strings.EqualFold(receivedSign, expectedSign) {
		return nil
	}

	return fmt.Errorf("signature mismatch: expected %s, got %s", expectedSign, receivedSign)
}

// signMD5 computes MD5 signature and returns uppercase hex string
func signMD5(data string) string {
	hash := md5.Sum([]byte(data))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// signHMACSHA256 computes HMAC-SHA256 signature and returns lowercase hex string
func signHMACSHA256(data string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
