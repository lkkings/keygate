# Payment Integration Troubleshooting Guide

## Task 19.3: Common Payment Issues and Solutions

This guide covers common issues encountered when integrating payment providers.

---

## Signature Verification Failures

### Stripe Signature Mismatch

**Symptom**: Webhook returns `400 Bad Request` with "Invalid signature"

**Causes**:
1. Incorrect webhook secret configured
2. Raw request body modified before verification
3. Using wrong Stripe account's webhook secret

**Solutions**:
1. Verify webhook secret matches Stripe dashboard:
   - Go to Stripe Dashboard > Webhooks
   - Click on your webhook endpoint
   - Show signing secret
   - Compare with KeyGate settings

2. Check server configuration:
   - Ensure raw body is passed to verification
   - Don't parse JSON before verification
   - Middleware shouldn't modify request body

3. Test with Stripe CLI:
   ```bash
   stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe
   stripe trigger payment_intent.succeeded
   ```

---

### Alipay Signature Verification Error

**Symptom**: `sign_check_error` or signature verification failed

**Causes**:
1. Using wrong Alipay public key
2. Private key format incorrect
3. Character encoding issues

**Solutions**:
1. Verify public key configuration:
   - Download Alipay public key from console
   - Ensure it's in PEM format
   - Remove line breaks if copying from web

2. Check private key format:
   ```
   -----BEGIN RSA PRIVATE KEY-----
   [Your key content]
   -----END RSA PRIVATE KEY-----
   ```

3. Verify character encoding:
   - Use UTF-8 for all strings
   - Don't URL-decode parameters before verification

4. Test in sandbox first:
   - Use Alipay sandbox environment
   - Complete test transaction
   - Review signature verification logs

---

### WeChat Pay Sign Error

**Symptom**: `sign error` in webhook response

**Causes**:
1. Incorrect API key
2. Parameter sorting error
3. Missing or wrong `key` parameter

**Solutions**:
1. Verify API key:
   - Check WeChat Pay merchant console
   - API key is 32 characters
   - Case-sensitive

2. Check signature algorithm:
   - Must include all parameters except `sign`
   - Sort alphabetically
   - Append `&key=API_KEY` at the end
   - MD5 hash must be uppercase

3. Test signature generation:
   ```
   Example: appid=wx123&mch_id=456&out_trade_no=789&key=YOUR_API_KEY
   MD5 → UPPERCASE
   ```

---

## Missing Currency Price

**Symptom**: Payment provider not available during checkout

**Cause**: Plan doesn't have price set for provider's currency

**Solutions**:
1. Check plan configuration:
   - Navigate to **Plans > Edit Plan**
   - Verify currency is set:
     - Stripe requires: `price_usd`
     - Alipay requires: `price_cny`
     - WeChat Pay requires: `price_cny`
     - ePay: depends on configuration

2. Update plan pricing:
   - Set price in required currency
   - Leave at 0 to hide from that provider
   - Save and retry checkout

3. Provider filtering logic:
   ```
   Stripe: available if price_usd > 0
   Alipay: available if price_cny > 0
   WeChat: available if price_cny > 0
   ```

---

## Webhook Timeout

**Symptom**: Provider retrying webhooks, "Timeout" in logs

**Causes**:
1. Slow database queries
2. External API calls in webhook handler
3. Server resource constraints

**Solutions**:
1. Optimize webhook processing:
   - Process asynchronously if possible
   - Return 200 response quickly
   - Handle errors gracefully

2. Check server logs:
   - Look for slow queries
   - Monitor CPU/memory usage
   - Scale resources if needed

3. Increase timeout (provider side):
   - Stripe: 30 seconds (not configurable)
   - Alipay: Contact support
   - WeChat Pay: Check merchant settings

---

## Duplicate Webhooks

**Symptom**: Multiple licenses created for same payment

**Expected Behavior**: KeyGate handles this automatically via idempotency

**Verification**:
1. Check transaction table:
   ```sql
   SELECT * FROM payment_transactions 
   WHERE provider_tx_id = 'txn_123';
   ```
   Should return only one row

2. If duplicates exist:
   - Indicates idempotency check bypassed
   - Check for unique constraint on `(provider_name, provider_tx_id)`
   - Review database migration

3. Manual cleanup:
   - Identify duplicate licenses
   - Keep earliest created license
   - Revoke duplicates
   - Refund if necessary

---

## Payment Stuck in Pending

**Symptom**: Payment shows as pending in provider dashboard but no webhook received

**Causes**:
1. Webhook URL incorrect
2. Webhook delivery failed
3. Provider delay in sending webhook

**Solutions**:
1. Verify webhook URL:
   - Check provider dashboard webhook settings
   - Ensure HTTPS
   - Verify URL is accessible publicly

2. Check webhook delivery logs:
   - **Stripe**: Dashboard > Webhooks > Recent deliveries
   - **Alipay**: Check notification logs
   - **WeChat Pay**: Contact support

3. Manual status check:
   - Use provider API to query payment status
   - Admin panel: **Transactions > Refresh Status**
   - May need to manually fulfill

---

## Refund Failures

**Symptom**: Refund request fails with error

**Common Errors**:

### "Charge already refunded"
- Check transaction status in admin panel
- Verify not already refunded
- Check provider dashboard

### "Insufficient balance"
- Stripe: Ensure account has funds
- May need to wait for settlement
- Contact provider support

### "Refund window expired"
- Some providers limit refund timeframe
- Alipay: 180 days
- Check provider-specific limits

**Solutions**:
1. Verify transaction is refundable:
   - Status must be "completed"
   - Within refund window
   - Not already refunded

2. Check refund amount:
   - Cannot exceed original amount
   - Partial refunds must be > 0

3. Manual refund:
   - Process refund through provider dashboard
   - Update transaction status in KeyGate manually
   - Document in transaction notes

---

## QR Code Not Displaying

**Symptom**: Alipay or WeChat Pay QR code doesn't show

**Causes**:
1. Missing `code_url` in response
2. Frontend rendering issue
3. Payment mode incorrect

**Solutions**:
1. Check API response:
   ```json
   {
     "checkout_url": null,
     "qr_code_url": "weixin://wxpay/bizpayurl?pr=ABC123",
     "mode": "native"
   }
   ```

2. Verify payment mode:
   - Alipay: Should use "Native" mode
   - WeChat: Should use "Native" mode
   - Check provider configuration

3. Frontend debugging:
   - Check browser console for errors
   - Verify QR code component receives URL
   - Test QR code URL directly

---

## Currency Conversion Issues

**Symptom**: Wrong amount shown during checkout

**Important**: KeyGate does NOT perform automatic currency conversion

**Solutions**:
1. Verify plan prices are set correctly:
   - USD price: in cents ($99.00 = 9900)
   - CNY price: in fen (¥688 = 68800)
   - HKD price: in cents (HK$777 = 77700)

2. Check provider currency:
   - Stripe: USD
   - Alipay: CNY
   - WeChat Pay: CNY
   - ePay: Multiple (configure per merchant)

3. Don't expect automatic conversion:
   - Set prices manually for each currency
   - Use realistic exchange rates
   - Round to convenient amounts

---

## Renewal Reminders Not Sending

**Symptom**: No renewal reminder emails

**Causes**:
1. Reminders disabled in settings
2. Cron job not running
3. License already expired

**Solutions**:
1. Check settings:
   - Admin panel: **Settings > Renewals**
   - Verify "renewal_reminders_enabled" is true

2. Verify cron job:
   - Check cron schedule is running
   - Run manually: `/api/v1/admin/renewal-reminders/send`
   - Check logs for errors

3. Test with specific license:
   - Set license to expire in 30 days
   - Run reminder job manually
   - Verify email sent

4. Check email configuration:
   - Verify SMTP settings
   - Test email service
   - Check spam folder

---

## Database Migration Failures

**Symptom**: Error during migration

**Solutions**:
1. Check migration status:
   ```bash
   go run cmd/migrate/main.go status
   ```

2. Run migrations:
   ```bash
   go run cmd/migrate/main.go up
   ```

3. Rollback if needed:
   ```bash
   go run cmd/migrate/main.go down
   ```

4. Manual fixes:
   - Backup database first
   - Check migration SQL
   - Apply manually if needed

---

## Provider-Specific Issues

### Stripe Issues
- **Docs**: https://stripe.com/docs
- **Support**: https://support.stripe.com
- **Status**: https://status.stripe.com

### Alipay Issues
- **Docs**: https://global.alipay.com/docs
- **Forum**: https://forum.alipay.com
- **Sandbox**: https://openhome.alipay.com/dev/

### WeChat Pay Issues
- **Docs**: https://pay.weixin.qq.com/wiki/doc/api/
- **Support**: Merchant console help center
- **Sandbox**: Apply through merchant console

---

## Getting Help

### KeyGate Support

1. **Check logs first**:
   - Admin panel: **Logs > Payment Events**
   - Server logs: `/var/log/keygate/`

2. **Gather information**:
   - Transaction ID
   - Error message
   - Timestamp
   - Provider name

3. **Create support ticket**:
   - Include all gathered information
   - Screenshots if applicable
   - Expected vs actual behavior

### Debug Mode

Enable debug logging:
1. Set `LOG_LEVEL=debug` in environment
2. Restart server
3. Reproduce issue
4. Check detailed logs

---

## Prevention

### Best Practices

1. **Test in sandbox first**
   - Complete full payment flow
   - Test webhooks
   - Verify license creation

2. **Monitor logs**
   - Set up alerts for errors
   - Review webhook failures daily
   - Track refund patterns

3. **Regular audits**
   - Reconcile transactions monthly
   - Check for stuck payments
   - Verify license counts

4. **Keep credentials secure**
   - Rotate keys periodically
   - Use environment variables
   - Never commit secrets to git

5. **Document custom configurations**
   - Note any non-standard settings
   - Document troubleshooting steps
   - Keep provider contact info handy
