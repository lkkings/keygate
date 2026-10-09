package wechat

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// SignRequest generates MD5 signature for WeChat Pay request
// Algorithm: Sort params alphabetically, concatenate key=value with &, append &key={api_key}, MD5, uppercase
func SignRequest(req interface{}, apiKey string) (string, error) {
	params := structToMap(req)
	return Sign(params, apiKey)
}

// Sign generates MD5 signature from parameter map
func Sign(params map[string]string, apiKey string) (string, error) {
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

	// Sort keys
	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build string to sign
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

	// Compute MD5 and convert to uppercase
	hash := md5.Sum([]byte(builder.String()))
	signature := strings.ToUpper(hex.EncodeToString(hash[:]))

	return signature, nil
}

// VerifySign verifies WeChat Pay webhook signature
func VerifySign(params map[string]string, apiKey string) error {
	receivedSign := params["sign"]
	if receivedSign == "" {
		return fmt.Errorf("missing sign field")
	}

	expectedSign, err := Sign(params, apiKey)
	if err != nil {
		return fmt.Errorf("failed to compute signature: %w", err)
	}

	if receivedSign != expectedSign {
		return fmt.Errorf("signature mismatch: expected %s, got %s", expectedSign, receivedSign)
	}

	return nil
}

// VerifyResponseSign verifies signature in XML response
func VerifyResponseSign(resp interface{}, apiKey string) error {
	params := structToMap(resp)
	return VerifySign(params, apiKey)
}

// structToMap converts a struct to map[string]string using XML tags
func structToMap(v interface{}) map[string]string {
	result := make(map[string]string)
	val := reflect.ValueOf(v)

	// Dereference pointer
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		typeField := typ.Field(i)

		// Get XML tag
		xmlTag := typeField.Tag.Get("xml")
		if xmlTag == "" || xmlTag == "-" || xmlTag == "xml" {
			continue
		}

		// Remove omitempty and other options
		tagParts := strings.Split(xmlTag, ",")
		key := tagParts[0]

		// Convert field value to string
		var value string
		switch field.Kind() {
		case reflect.String:
			value = field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() != 0 {
				value = fmt.Sprintf("%d", field.Int())
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if field.Uint() != 0 {
				value = fmt.Sprintf("%d", field.Uint())
			}
		case reflect.Float32, reflect.Float64:
			if field.Float() != 0 {
				value = fmt.Sprintf("%f", field.Float())
			}
		case reflect.Bool:
			if field.Bool() {
				value = "true"
			}
		}

		if value != "" {
			result[key] = value
		}
	}

	return result
}

// GenerateNonceStr generates a random nonce string
func GenerateNonceStr() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
