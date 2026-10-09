package alipay

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client handles Alipay API interactions
type Client struct {
	AppID          string
	PrivateKey     *rsa.PrivateKey
	AlipayPublicKey *rsa.PublicKey
	Gateway        string // https://openapi.alipay.com/gateway.do or sandbox
	HTTPClient     *http.Client
	SignType       string // RSA2 (default)
}

// Config holds Alipay client configuration
type Config struct {
	AppID             string
	PrivateKeyPEM     string // RSA private key in PEM format
	AlipayPublicKeyPEM string // Alipay public key in PEM format
	IsSandbox         bool
}

// NewClient creates a new Alipay client
func NewClient(config Config) (*Client, error) {
	// Parse private key
	privateKey, err := parsePrivateKey(config.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Parse Alipay public key
	publicKey, err := parsePublicKey(config.AlipayPublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Alipay public key: %w", err)
	}

	gateway := "https://openapi.alipay.com/gateway.do"
	if config.IsSandbox {
		gateway = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	}

	return &Client{
		AppID:          config.AppID,
		PrivateKey:     privateKey,
		AlipayPublicKey: publicKey,
		Gateway:        gateway,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
		SignType:       "RSA2",
	}, nil
}

// parsePrivateKey parses an RSA private key from PEM format
func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	// Remove PEM headers if present
	pemStr = strings.ReplaceAll(pemStr, "-----BEGIN RSA PRIVATE KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "-----END RSA PRIVATE KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "-----BEGIN PRIVATE KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "-----END PRIVATE KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "\n", "")
	pemStr = strings.ReplaceAll(pemStr, "\r", "")
	pemStr = strings.TrimSpace(pemStr)

	data, err := base64.StdEncoding.DecodeString(pemStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Try parsing as PKCS1
	privateKey, err := x509.ParsePKCS1PrivateKey(data)
	if err == nil {
		return privateKey, nil
	}

	// Try parsing as PKCS8
	key, err := x509.ParsePKCS8PrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return rsaKey, nil
}

// parsePublicKey parses an RSA public key from PEM format
func parsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	// Remove PEM headers if present
	pemStr = strings.ReplaceAll(pemStr, "-----BEGIN PUBLIC KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "-----END PUBLIC KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "\n", "")
	pemStr = strings.ReplaceAll(pemStr, "\r", "")
	pemStr = strings.TrimSpace(pemStr)

	data, err := base64.StdEncoding.DecodeString(pemStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	pub, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// UnifiedOrder creates a payment order (alipay.trade.page.pay)
func (c *Client) UnifiedOrder(ctx context.Context, params map[string]interface{}) (*TradePagePayResponse, error) {
	bizContent, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal biz_content: %w", err)
	}

	reqParams := map[string]string{
		"app_id":      c.AppID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   c.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizContent),
	}

	// Sign the request
	sign, err := c.Sign(reqParams)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	reqParams["sign"] = sign

	// Build URL with query parameters
	redirectURL := c.Gateway + "?" + buildQuery(reqParams)

	return &TradePagePayResponse{
		RedirectURL: redirectURL,
		OutTradeNo:  params["out_trade_no"].(string),
	}, nil
}

// QueryOrder queries order status (alipay.trade.query)
func (c *Client) QueryOrder(ctx context.Context, outTradeNo string) (*TradeQueryResponse, error) {
	bizContent, _ := json.Marshal(map[string]string{
		"out_trade_no": outTradeNo,
	})

	reqParams := map[string]string{
		"app_id":      c.AppID,
		"method":      "alipay.trade.query",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   c.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizContent),
	}

	sign, err := c.Sign(reqParams)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	reqParams["sign"] = sign

	// Make HTTP request
	resp, err := c.HTTPClient.PostForm(c.Gateway, urlValuesFromMap(reqParams))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		TradeQueryResponse TradeQueryResponse `json:"alipay_trade_query_response"`
		Sign               string            `json:"sign"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Verify response signature
	if err := c.VerifySign(result.TradeQueryResponse, result.Sign); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &result.TradeQueryResponse, nil
}

// RefundOrder initiates a refund (alipay.trade.refund)
func (c *Client) RefundOrder(ctx context.Context, outTradeNo string, refundAmount float64, refundReason string) (*TradeRefundResponse, error) {
	bizContent, _ := json.Marshal(map[string]interface{}{
		"out_trade_no":   outTradeNo,
		"refund_amount":  refundAmount,
		"refund_reason":  refundReason,
	})

	reqParams := map[string]string{
		"app_id":      c.AppID,
		"method":      "alipay.trade.refund",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   c.SignType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizContent),
	}

	sign, err := c.Sign(reqParams)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	reqParams["sign"] = sign

	resp, err := c.HTTPClient.PostForm(c.Gateway, urlValuesFromMap(reqParams))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		TradeRefundResponse TradeRefundResponse `json:"alipay_trade_refund_response"`
		Sign               string              `json:"sign"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result.TradeRefundResponse, nil
}

// Sign signs the request parameters using RSA2
func (c *Client) Sign(params map[string]string) (string, error) {
	return SignWithRSA2(params, c.PrivateKey)
}

// VerifySign verifies Alipay's response signature
func (c *Client) VerifySign(data interface{}, sign string) error {
	// Convert data to JSON string
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return VerifyWithRSA2(string(jsonData), sign, c.AlipayPublicKey)
}

// buildQuery builds URL query string from params
func buildQuery(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}

// urlValuesFromMap converts map to url.Values
func urlValuesFromMap(params map[string]string) url.Values {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values
}

// Response types

type TradePagePayResponse struct {
	RedirectURL string
	OutTradeNo  string
}

type TradeQueryResponse struct {
	Code       string `json:"code"`
	Msg        string `json:"msg"`
	TradeNo    string `json:"trade_no"`
	OutTradeNo string `json:"out_trade_no"`
	TradeStatus string `json:"trade_status"` // TRADE_SUCCESS, TRADE_FINISHED, TRADE_CLOSED, WAIT_BUYER_PAY
	TotalAmount string `json:"total_amount"`
	BuyerLogonID string `json:"buyer_logon_id"`
}

type TradeRefundResponse struct {
	Code          string `json:"code"`
	Msg           string `json:"msg"`
	TradeNo       string `json:"trade_no"`
	OutTradeNo    string `json:"out_trade_no"`
	RefundFee     string `json:"refund_fee"`
	GMTRefundPay  string `json:"gmt_refund_pay"`
}
