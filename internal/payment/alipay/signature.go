package alipay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

// SignWithRSA2 signs parameters using SHA256WithRSA (Alipay's RSA2)
// Following Alipay specification:
// 1. Sort parameters by key (ascending, lexicographically)
// 2. Build sign content as key1=value1&key2=value2&...
// 3. Sign with SHA256WithRSA
// 4. Base64 encode the signature
func SignWithRSA2(params map[string]string, privateKey *rsa.PrivateKey) (string, error) {
	// Build sign content
	content := buildSignContent(params)

	// Hash the content
	hashed := sha256.Sum256([]byte(content))

	// Sign with RSA
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// Base64 encode
	return base64.StdEncoding.EncodeToString(signature), nil
}

// buildSignContent builds the string to be signed from parameters
// Parameters are sorted by key and joined with &
// Empty values and "sign" key are excluded
func buildSignContent(params map[string]string) string {
	// Get sorted keys (excluding "sign" and empty values)
	var keys []string
	for k, v := range params {
		if k != "sign" && v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Build the content string
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, k+"="+params[k])
	}

	return strings.Join(pairs, "&")
}

// VerifyWithRSA2 verifies an Alipay response signature
func VerifyWithRSA2(content string, sign string, publicKey *rsa.PublicKey) error {
	// Decode base64 signature
	signatureBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Hash the content
	hashed := sha256.Sum256([]byte(content))

	// Verify signature
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signatureBytes)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// VerifyNotifySign verifies an async notification signature from Alipay
// Notifications come as form parameters, so we need to rebuild the sign content
func VerifyNotifySign(params map[string]string, publicKey *rsa.PublicKey) error {
	// Extract the signature
	sign, ok := params["sign"]
	if !ok {
		return fmt.Errorf("missing sign parameter")
	}

	// Build content (excluding sign and sign_type)
	filteredParams := make(map[string]string)
	for k, v := range params {
		if k != "sign" && k != "sign_type" && v != "" {
			filteredParams[k] = v
		}
	}

	content := buildSignContent(filteredParams)

	return VerifyWithRSA2(content, sign, publicKey)
}
