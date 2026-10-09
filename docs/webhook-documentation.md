# Payment Webhook Documentation

## Task 19.2: Webhook URLs and Signature Verification

This document details webhook endpoints, signature verification, and payload formats for each payment provider.

---

## Webhook URLs

All webhook endpoints follow the pattern: `https://your-domain.com/api/v1/webhooks/{provider}`

| Provider | Webhook URL | Method |
|----------|-------------|--------|
| Stripe | `/api/v1/webhooks/stripe` | POST |
| Alipay | `/api/v1/webhooks/alipay` | POST |
| WeChat Pay | `/api/v1/webhooks/wechat` | POST |
| ePay | `/api/v1/webhooks/epay` | POST |

---

## Stripe Webhooks

### Signature Verification

Stripe uses HMAC SHA-256 signatures sent in the `Stripe-Signature` header.

**Header Format**:
```
Stripe-Signature: t=1234567890,v1=signature_hash
```

**Verification Process**:
1. Extract timestamp and signature from header
2. Construct signed payload: `timestamp.raw_body`
3. Compute HMAC SHA-256 using webhook secret
4. Compare computed signature with received signature
5. Verify timestamp is within 5 minutes (replay attack prevention)

**Library**: KeyGate uses `stripe.ConstructEvent()` for automatic verification.

### Required Events

Subscribe to these events in Stripe Dashboard:

- `payment_intent.succeeded` - Payment completed successfully
- `payment_intent.payment_failed` - Payment failed
- `charge.refunded` - Refund processed

### Payload Example

```json
{
  "id": "evt_1234567890",
  "type": "payment_intent.succeeded",
  "data": {
    "object": {
      "id": "pi_1234567890",
      "amount": 9900,
      "currency": "usd",
      "status": "succeeded",
      "metadata": {
        "plan_id": "plan_abc",
        "customer_email": "user@example.com"
      }
    }
  }
}
```

---

## Alipay Webhooks

### Signature Verification

Alipay uses RSA-SHA256 signatures with their private key.

**Verification Process**:
1. Extract `sign` parameter from POST body
2. Remove `sign` and `sign_type` from parameters
3. Sort remaining parameters alphabetically
4. Concatenate as `key1=value1&key2=value2`
5. Verify signature using Alipay's public key

**Algorithm**: RSA with SHA-256

### Required Parameters

The webhook payload includes these key parameters:

- `app_id` - Your application ID
- `trade_no` - Alipay transaction ID
- `out_trade_no` - Your order ID
- `trade_status` - Payment status
- `total_amount` - Transaction amount
- `buyer_email` - Customer email
- `sign` - RSA signature
- `sign_type` - Signature algorithm (RSA2)

### Trade Status Values

- `TRADE_SUCCESS` - Payment completed (final status)
- `TRADE_FINISHED` - Transaction finished (after refund period)
- `TRADE_CLOSED` - Transaction closed without payment

### Payload Example

```
app_id=2021001234567890&
buyer_email=user@example.com&
out_trade_no=ORDER20240101123456&
trade_no=2024010122001234567890&
trade_status=TRADE_SUCCESS&
total_amount=99.00&
sign=ERITJKEIJKJHKKKKKKKHJEREEEEEEEEEEE&
sign_type=RSA2
```

---

## WeChat Pay Webhooks

### Signature Verification

WeChat Pay uses MD5 signature with API key.

**Verification Process**:
1. Parse XML payload
2. Extract all parameters except `sign`
3. Sort parameters alphabetically
4. Concatenate as `key1=value1&key2=value2&key=API_KEY`
5. Compute MD5 hash (uppercase)
6. Compare with received `sign` field

**Response Format**: Must reply with XML

```xml
<xml>
  <return_code><![CDATA[SUCCESS]]></return_code>
  <return_msg><![CDATA[OK]]></return_msg>
</xml>
```

### Required Fields

- `return_code` - SUCCESS or FAIL
- `return_msg` - Result message
- `appid` - Application ID
- `mch_id` - Merchant ID
- `out_trade_no` - Order ID
- `transaction_id` - WeChat transaction ID
- `total_fee` - Amount in cents
- `sign` - MD5 signature

### Payload Example

```xml
<xml>
  <return_code><![CDATA[SUCCESS]]></return_code>
  <return_msg><![CDATA[OK]]></return_msg>
  <appid><![CDATA[wx1234567890abcdef]]></appid>
  <mch_id><![CDATA[1234567890]]></mch_id>
  <out_trade_no><![CDATA[ORDER20240101123456]]></out_trade_no>
  <transaction_id><![CDATA[4200001234567890]]></transaction_id>
  <total_fee>9900</total_fee>
  <sign><![CDATA[A1B2C3D4E5F6...]]></sign>
</xml>
```

---

## ePay Webhooks

### Signature Verification

ePay uses HMAC SHA-256 with API secret.

**Verification Process**:
1. Extract signature from `X-ePay-Signature` header
2. Concatenate request body with secret
3. Compute HMAC SHA-256
4. Compare with received signature

### Payload Format

JSON format similar to Stripe:

```json
{
  "event": "payment.completed",
  "transaction_id": "epay_1234567890",
  "amount": 9900,
  "currency": "USD",
  "status": "completed",
  "merchant_order_id": "ORDER123",
  "customer_email": "user@example.com"
}
```

---

## Security Requirements

### HTTPS Only

All webhook URLs **must** use HTTPS in production:
- Prevents man-in-the-middle attacks
- Required by most payment providers
- Use valid SSL certificate (not self-signed)

### IP Whitelisting (Optional)

Some providers publish webhook source IPs:
- **Stripe**: No IP whitelist needed (signature verification is sufficient)
- **Alipay**: Check documentation for IP ranges
- **WeChat Pay**: May require IP whitelist in merchant console
- **ePay**: Contact support for IP ranges

### Rate Limiting

Implement rate limiting on webhook endpoints:
- Prevents DoS attacks
- KeyGate includes built-in rate limiting
- Recommended: 100 requests per minute per IP

---

## Webhook Retry Logic

### Provider Behavior

- **Stripe**: Retries for up to 3 days with exponential backoff
- **Alipay**: Retries 8 times over 24 hours
- **WeChat Pay**: Retries for 8 hours
- **ePay**: Depends on configuration

### Response Codes

Return appropriate HTTP status codes:

| Code | Meaning | Provider Action |
|------|---------|-----------------|
| 200 | Success | Stop retrying |
| 4xx | Client error | Stop retrying (permanent failure) |
| 5xx | Server error | Retry with backoff |

---

## Testing Webhooks

### Stripe

Use Stripe CLI for local testing:

```bash
stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe
stripe trigger payment_intent.succeeded
```

### Alipay

Use Alipay sandbox:
1. Complete test payment in sandbox
2. Monitor webhook delivery in sandbox console
3. View request/response logs

### WeChat Pay

Contact WeChat Pay support for sandbox webhook testing.

### Local Testing

Use tools like ngrok for local webhook testing:

```bash
ngrok http 8080
# Use the ngrok URL in provider webhook settings
```

---

## Monitoring and Debugging

### Webhook Logs

KeyGate logs all webhook events:
- Admin panel: **Logs > Webhook Events**
- Includes: timestamp, provider, event type, status
- Raw payload available for failed webhooks

### Common Issues

See [Troubleshooting Guide](./troubleshooting.md) for:
- Signature verification failures
- Timeout errors
- Duplicate events
- Missing webhooks

---

## Idempotency

KeyGate implements idempotency using `(provider, transaction_id)` uniqueness:
- Duplicate webhooks are safely ignored
- Prevents duplicate license creation
- Transaction table enforces uniqueness constraint

---

## Support

For webhook issues:
1. Check webhook logs in admin panel
2. Verify signature verification settings
3. Test with provider's sandbox
4. Contact provider support with transaction ID
