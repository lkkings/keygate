package alipay

import (
	"net/url"
	"strings"
)

// EncodeParams encodes parameters for URL query string
// Following Alipay specification for parameter encoding
func EncodeParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	var pairs []string
	for key, value := range params {
		if value == "" {
			continue
		}
		encodedKey := url.QueryEscape(key)
		encodedValue := url.QueryEscape(value)
		pairs = append(pairs, encodedKey+"="+encodedValue)
	}

	return strings.Join(pairs, "&")
}

// DecodeParams decodes URL query string to parameters map
func DecodeParams(queryString string) (map[string]string, error) {
	values, err := url.ParseQuery(queryString)
	if err != nil {
		return nil, err
	}

	params := make(map[string]string)
	for key, value := range values {
		if len(value) > 0 {
			params[key] = value[0]
		}
	}

	return params, nil
}

// ParseNotifyParams parses form-encoded notification parameters from Alipay
func ParseNotifyParams(formData map[string][]string) map[string]string {
	params := make(map[string]string)
	for key, values := range formData {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}
