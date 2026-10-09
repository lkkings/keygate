# Multi-Payment Gateway Integration - Admin Guide

## Task 19.1: Configuring Payment Providers

This guide covers how to set up and configure each payment provider in KeyGate.

## Prerequisites

- Admin access to KeyGate
- Payment provider accounts (Stripe, Alipay, WeChat Pay, or ePay)
- API credentials from each provider

---

## Stripe Configuration

### 1. Obtain Credentials

1. Go to [Stripe Dashboard](https://dashboard.stripe.com)
2. Navigate to **Developers > API keys**
3. Copy your **Publishable key** and **Secret key**
4. Navigate to **Developers > Webhooks**
5. Create a webhook endpoint (see Webhook Setup below)
6. Copy the **Webhook signing secret**

### 2. Configure in KeyGate

1. Log in to KeyGate admin panel
2. Navigate to **Settings > Payment Providers**
3. Enable **Stripe**
4. Enter the following settings:
   - **Secret Key**: `sk_live_...` or `sk_test_...`
   - **Publishable Key**: `pk_live_...` or `pk_test_...`
   - **Webhook Secret**: `whsec_...`
5. Click **Save**

### 3. Webhook Setup

**Webhook URL**: `https://your-domain.com/api/v1/webhooks/stripe`

**Events to subscribe to**:
- `payment_intent.succeeded`
- `payment_intent.payment_failed`
- `charge.refunded`

---

## Alipay Configuration

### 1. Obtain Credentials

1. Register at [Alipay Open Platform](https://open.alipay.com)
2. Create an application
3. Generate RSA key pair (2048-bit recommended)
4. Upload public key to Alipay console
5. Download Alipay's public key
6. Obtain your **App ID**

### 2. Configure in KeyGate

1. Log in to KeyGate admin panel
2. Navigate to **Settings > Payment Providers**
3. Enable **Alipay**
4. Enter the following settings:
   - **App ID**: Your Alipay App ID
   - **Private Key**: Your RSA private key (PEM format)
   - **Alipay Public Key**: Alipay's public key
   - **Gateway**: `https://openapi.alipay.com/gateway.do` (production) or sandbox URL
5. Click **Save**

### 3. Webhook Setup (Async Notification)

**Webhook URL**: `https://your-domain.com/api/v1/webhooks/alipay`

Configure in Alipay console:
- Set notification URL for your application
- Alipay will POST payment status updates to this URL
- Must use HTTPS in production

---

## WeChat Pay Configuration

### 1. Obtain Credentials

1. Register at [WeChat Pay Merchant Platform](https://pay.weixin.qq.com)
2. Complete merchant verification
3. Obtain the following from merchant console:
   - **Merchant ID** (mch_id)
   - **App ID** (for mobile/web app)
   - **API Key** (32-character key for signature)
   - **API Certificate** (for refunds, optional)

### 2. Configure in KeyGate

1. Log in to KeyGate admin panel
2. Navigate to **Settings > Payment Providers**
3. Enable **WeChat Pay**
4. Enter the following settings:
   - **App ID**: Your WeChat App ID
   - **Merchant ID**: Your merchant ID (mch_id)
   - **API Key**: 32-character API key
   - **API Certificate** (optional): For refund operations
5. Click **Save**

### 3. Webhook Setup

**Webhook URL**: `https://your-domain.com/api/v1/webhooks/wechat`

Configure in WeChat Pay console:
- Set payment notification URL
- Must use HTTPS
- IP whitelist may be required

---

## ePay Configuration

### 1. Obtain Credentials

1. Contact ePay for merchant account
2. Receive API credentials from ePay support
3. Obtain:
   - **Merchant ID**
   - **API Key**
   - **API Secret**

### 2. Configure in KeyGate

1. Log in to KeyGate admin panel
2. Navigate to **Settings > Payment Providers**
3. Enable **ePay**
4. Enter the following settings:
   - **Merchant ID**: Your ePay merchant ID
   - **API Key**: Your API key
   - **API Secret**: Your API secret
   - **Gateway URL**: Provided by ePay
5. Click **Save**

---

## Testing Configuration

### Sandbox Mode

Most providers offer sandbox/test environments:

- **Stripe**: Use test API keys (starting with `sk_test_` and `pk_test_`)
- **Alipay**: Use sandbox gateway `https://openapi.alipaydev.com/gateway.do`
- **WeChat Pay**: Apply for sandbox merchant account
- **ePay**: Contact support for test credentials

### Verification Steps

1. Create a test plan with pricing in relevant currencies
2. Initiate a checkout session
3. Complete payment using test credentials
4. Verify webhook is received
5. Confirm license is created
6. Check transaction appears in admin panel

---

## Multi-Currency Setup

### Setting Plan Prices

For each plan, you can set prices in multiple currencies:

1. Navigate to **Plans > Edit Plan**
2. Set prices for each currency:
   - **USD**: For Stripe, ePay
   - **CNY**: For Alipay, WeChat Pay
   - **HKD**: For ePay (Hong Kong)
3. Leave a currency blank if not supported for that plan
4. Payment providers will be filtered based on available currencies

### Currency Conversion

- KeyGate does not perform automatic currency conversion
- Set prices manually for each currency
- Providers only see the currency they support

---

## Security Best Practices

1. **Use HTTPS**: All webhook URLs must use HTTPS in production
2. **Verify Signatures**: Always verify webhook signatures (automatic in KeyGate)
3. **Rotate Keys**: Periodically rotate API keys and secrets
4. **Environment Separation**: Use separate credentials for test/production
5. **Least Privilege**: Use restricted API keys when possible
6. **Monitor Logs**: Regularly review payment logs for anomalies

---

## Common Issues

See the [Troubleshooting Guide](./troubleshooting.md) for solutions to common problems.

---

## Support

For payment provider-specific issues:
- **Stripe**: [Stripe Support](https://support.stripe.com)
- **Alipay**: [Alipay Docs](https://global.alipay.com/docs)
- **WeChat Pay**: [WeChat Pay Docs](https://pay.weixin.qq.com/index.php/public/wechatpay)
- **ePay**: Contact your ePay account manager

For KeyGate issues:
- Check logs in admin panel under **Logs > Payment Events**
- Contact KeyGate support with transaction ID
