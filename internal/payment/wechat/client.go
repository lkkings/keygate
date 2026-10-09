package wechat

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles WeChat Pay API interactions
type Client struct {
	AppID      string
	MerchantID string
	APIKey     string
	Gateway    string // https://api.mch.weixin.qq.com or sandbox
	HTTPClient *http.Client
	CertFile   string // Path to apiclient_cert.pem (for refunds)
	KeyFile    string // Path to apiclient_key.pem (for refunds)
}

// Config holds WeChat Pay client configuration
type Config struct {
	AppID      string
	MerchantID string
	APIKey     string
	IsSandbox  bool
	CertFile   string // Optional: for refund operations
	KeyFile    string // Optional: for refund operations
}

// NewClient creates a new WeChat Pay client
func NewClient(config Config) (*Client, error) {
	gateway := "https://api.mch.weixin.qq.com"
	if config.IsSandbox {
		gateway = "https://api.mch.weixin.qq.com/sandboxnew"
	}

	return &Client{
		AppID:      config.AppID,
		MerchantID: config.MerchantID,
		APIKey:     config.APIKey,
		Gateway:    gateway,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		CertFile:   config.CertFile,
		KeyFile:    config.KeyFile,
	}, nil
}

// UnifiedOrder creates a unified order (统一下单)
// Supports NATIVE (QR code), JSAPI (in-app), and APP payment modes
func (c *Client) UnifiedOrder(ctx context.Context, params *UnifiedOrderRequest) (*UnifiedOrderResponse, error) {
	// Generate nonce_str
	params.NonceStr = GenerateNonceStr()
	params.AppID = c.AppID
	params.MchID = c.MerchantID

	// Sign the request
	sign, err := SignRequest(params, c.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	params.Sign = sign

	// Marshal to XML
	xmlData, err := xml.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	url := c.Gateway + "/pay/unifiedorder"
	resp, err := c.HTTPClient.Post(url, "application/xml", bytesReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result UnifiedOrderResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check return code
	if result.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("unified order failed: %s - %s", result.ReturnCode, result.ReturnMsg)
	}

	if result.ResultCode != "SUCCESS" {
		return nil, fmt.Errorf("unified order failed: %s - %s", result.ErrCode, result.ErrCodeDes)
	}

	// Verify response signature
	if err := VerifyResponseSign(&result, c.APIKey); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &result, nil
}

// QueryOrder queries order status (查询订单)
func (c *Client) QueryOrder(ctx context.Context, outTradeNo string) (*QueryOrderResponse, error) {
	req := &QueryOrderRequest{
		AppID:      c.AppID,
		MchID:      c.MerchantID,
		OutTradeNo: outTradeNo,
		NonceStr:   GenerateNonceStr(),
	}

	// Sign the request
	sign, err := SignRequest(req, c.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	req.Sign = sign

	// Marshal to XML
	xmlData, err := xml.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	url := c.Gateway + "/pay/orderquery"
	resp, err := c.HTTPClient.Post(url, "application/xml", bytesReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result QueryOrderResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check return code
	if result.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("query order failed: %s - %s", result.ReturnCode, result.ReturnMsg)
	}

	return &result, nil
}

// RefundOrder initiates a refund (申请退款)
// Requires merchant certificate for mutual TLS authentication
func (c *Client) RefundOrder(ctx context.Context, params *RefundRequest) (*RefundResponse, error) {
	// Load certificate for refund operations
	if c.CertFile == "" || c.KeyFile == "" {
		return nil, fmt.Errorf("certificate required for refund operations")
	}

	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Create HTTPS client with certificate
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	certClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Generate nonce_str and set fields
	params.NonceStr = GenerateNonceStr()
	params.AppID = c.AppID
	params.MchID = c.MerchantID

	// Sign the request
	sign, err := SignRequest(params, c.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	params.Sign = sign

	// Marshal to XML
	xmlData, err := xml.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTPS request with certificate
	url := c.Gateway + "/secapi/pay/refund"
	resp, err := certClient.Post(url, "application/xml", bytesReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result RefundResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check return code
	if result.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("refund failed: %s - %s", result.ReturnCode, result.ReturnMsg)
	}

	if result.ResultCode != "SUCCESS" {
		return nil, fmt.Errorf("refund failed: %s - %s", result.ErrCode, result.ErrCodeDes)
	}

	return &result, nil
}

// VerifySign verifies WeChat Pay webhook signature
func (c *Client) VerifySign(params map[string]string) error {
	return VerifySign(params, c.APIKey)
}

// Request and Response types

// UnifiedOrderRequest represents a unified order request
type UnifiedOrderRequest struct {
	XMLName        xml.Name `xml:"xml"`
	AppID          string   `xml:"appid"`
	MchID          string   `xml:"mch_id"`
	NonceStr       string   `xml:"nonce_str"`
	Sign           string   `xml:"sign"`
	Body           string   `xml:"body"`           // Product description
	OutTradeNo     string   `xml:"out_trade_no"`   // Merchant order ID
	TotalFee       int      `xml:"total_fee"`      // Amount in cents (分)
	SpbillCreateIP string   `xml:"spbill_create_ip"` // Client IP
	NotifyURL      string   `xml:"notify_url"`     // Webhook URL
	TradeType      string   `xml:"trade_type"`     // NATIVE, JSAPI, or APP
	OpenID         string   `xml:"openid,omitempty"` // Required for JSAPI
}

// UnifiedOrderResponse represents a unified order response
type UnifiedOrderResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"` // SUCCESS or FAIL
	ReturnMsg  string   `xml:"return_msg"`
	AppID      string   `xml:"appid"`
	MchID      string   `xml:"mch_id"`
	NonceStr   string   `xml:"nonce_str"`
	Sign       string   `xml:"sign"`
	ResultCode string   `xml:"result_code"` // SUCCESS or FAIL
	ErrCode    string   `xml:"err_code,omitempty"`
	ErrCodeDes string   `xml:"err_code_des,omitempty"`
	TradeType  string   `xml:"trade_type"`
	PrepayID   string   `xml:"prepay_id"`   // For JSAPI/APP
	CodeURL    string   `xml:"code_url"`    // For NATIVE (QR code)
}

// QueryOrderRequest represents an order query request
type QueryOrderRequest struct {
	XMLName      xml.Name `xml:"xml"`
	AppID        string   `xml:"appid"`
	MchID        string   `xml:"mch_id"`
	OutTradeNo   string   `xml:"out_trade_no"`
	NonceStr     string   `xml:"nonce_str"`
	Sign         string   `xml:"sign"`
}

// QueryOrderResponse represents an order query response
type QueryOrderResponse struct {
	XMLName       xml.Name `xml:"xml"`
	ReturnCode    string   `xml:"return_code"`
	ReturnMsg     string   `xml:"return_msg"`
	AppID         string   `xml:"appid"`
	MchID         string   `xml:"mch_id"`
	NonceStr      string   `xml:"nonce_str"`
	Sign          string   `xml:"sign"`
	ResultCode    string   `xml:"result_code"`
	OutTradeNo    string   `xml:"out_trade_no"`
	TransactionID string   `xml:"transaction_id"`
	TradeState    string   `xml:"trade_state"` // SUCCESS, REFUND, NOTPAY, CLOSED, etc.
	TotalFee      int      `xml:"total_fee"`
	TimeEnd       string   `xml:"time_end,omitempty"`
}

// RefundRequest represents a refund request
type RefundRequest struct {
	XMLName      xml.Name `xml:"xml"`
	AppID        string   `xml:"appid"`
	MchID        string   `xml:"mch_id"`
	NonceStr     string   `xml:"nonce_str"`
	Sign         string   `xml:"sign"`
	OutTradeNo   string   `xml:"out_trade_no"`
	OutRefundNo  string   `xml:"out_refund_no"` // Unique refund ID
	TotalFee     int      `xml:"total_fee"`     // Original amount
	RefundFee    int      `xml:"refund_fee"`    // Refund amount
	RefundDesc   string   `xml:"refund_desc,omitempty"`
}

// RefundResponse represents a refund response
type RefundResponse struct {
	XMLName       xml.Name `xml:"xml"`
	ReturnCode    string   `xml:"return_code"`
	ReturnMsg     string   `xml:"return_msg"`
	ResultCode    string   `xml:"result_code"`
	ErrCode       string   `xml:"err_code,omitempty"`
	ErrCodeDes    string   `xml:"err_code_des,omitempty"`
	AppID         string   `xml:"appid"`
	MchID         string   `xml:"mch_id"`
	NonceStr      string   `xml:"nonce_str"`
	Sign          string   `xml:"sign"`
	TransactionID string   `xml:"transaction_id"`
	OutTradeNo    string   `xml:"out_trade_no"`
	OutRefundNo   string   `xml:"out_refund_no"`
	RefundID      string   `xml:"refund_id"`
	RefundFee     int      `xml:"refund_fee"`
}

// Helper functions

func bytesReader(data []byte) io.Reader {
	return io.NopCloser(newBytesBuffer(data))
}

func newBytesBuffer(data []byte) *bytesBuffer {
	return &bytesBuffer{data: data}
}

type bytesBuffer struct {
	data []byte
	pos  int
}

func (b *bytesBuffer) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}
