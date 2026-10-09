# 多支付网关集成架构说明

## 📐 架构设计

### 分层架构

```
┌─────────────────────────────────────────────────────┐
│                   HTTP Handler                       │
│             (handler/payment/handler.go)            │
│  - 路由处理                                          │
│  - 请求验证                                          │
│  - 响应格式化                                        │
└────────────────────┬────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────┐
│                Payment Service                       │
│            (service/payment.go)                     │
│  - 业务逻辑                                          │
│  - License Fulfillment                               │
│  - 事务管理                                          │
│  - Email 和 Webhook 触发                            │
└────────────────────┬────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────┐
│              Payment Provider Registry               │
│            (payment/registry.go)                    │
│  - Provider 工厂                                     │
│  - 动态加载                                          │
└──────┬─────────────────────────────────────┬────────┘
       │                                     │
┌──────▼──────────┐                 ┌───────▼─────────┐
│ Alipay Provider │                 │ Stripe Provider │
│ (alipay/)       │                 │ (stripe/)       │
│                 │                 │                 │
│ - API 客户端    │                 │ - Stripe SDK    │
│ - RSA2 签名     │                 │ - Webhook 验证  │
│ - 沙箱支持      │                 │ - 退款          │
└─────────────────┘                 └─────────────────┘
       │                                     │
┌──────▼─────────────────────────────────────▼────────┐
│                   Data Store                         │
│            (store/payment_transaction.go)           │
│  - Payment Transactions                             │
│  - Licenses                                         │
│  - Plans                                            │
└─────────────────────────────────────────────────────┘
```

## 🔑 核心组件

### 1. PaymentProvider 接口

所有支付提供商的统一接口：

```go
type PaymentProvider interface {
    Name() string
    CreateCheckout(ctx, params) (CheckoutResult, error)
    VerifyWebhook(ctx, payload, signature) (WebhookEvent, error)
    RefundPayment(ctx, txID, amount, reason) (RefundResult, error)
    GetPaymentStatus(ctx, txID) (PaymentStatus, error)
}
```

### 2. PaymentService (业务逻辑层)

**职责:**
- 路由支付请求到正确的 provider
- 处理 webhook 事件
- **License Fulfillment** - 支付成功后自动创建 license
- 事务持久化
- 触发 email 和 webhook 通知

**关键方法:**
```go
func (s *PaymentService) HandleWebhook(ctx, provider, payload, sig) error
func (s *PaymentService) handleCompletedPayment(ctx, provider, event) error
func (s *PaymentService) RefundPayment(ctx, txID, amount, reason) (RefundResult, error)
```

**License Fulfillment 流程:**

1. **Webhook 接收** → `HandleWebhook()`
2. **Provider 验证** → `provider.VerifyWebhook()` 
3. **状态判断** → `switch event.Status`
4. **完成支付** → `handleCompletedPayment()`
   - 幂等性检查 (防止重复处理)
   - 提取 plan_id 和 product_id from metadata
   - 查询 plan 详情
   - 创建 License:
     - Perpetual: 无过期时间
     - Subscription: 根据 billing_interval 计算过期
     - Trial: 根据 trial_days 计算过期
   - 保存 PaymentTransaction 记录
   - 发送欢迎邮件 (含 license key)
   - 触发 `license.created` webhook

### 3. Provider Registry (工厂模式)

**自动注册:**
```go
// internal/payment/alipay/register.go
func init() {
    payment.RegisterProvider("alipay", func() payment.PaymentProvider {
        return NewProvider()
    })
}
```

**获取 provider:**
```go
provider, err := payment.GetProvider("alipay")
```

### 4. Handler 层

**路由:**
- `POST /api/v1/payment/:provider/checkout` - 创建支付会话
- `POST /api/v1/payment/:provider/webhook` - Webhook 接收
- `GET /api/v1/payment/:provider/return` - 同步回调
- `GET /api/v1/payment/:provider/status/:tx_id` - 查询状态
- `GET /api/v1/payment/providers` - 列出可用 providers

**Admin 路由:**
- `POST /api/v1/admin/payment/:provider/refund/:tx_id` - 退款
- `GET /api/v1/admin/payment/transactions/:tx_id` - 查询交易
- `GET /api/v1/admin/payment/transactions` - 交易列表

## 🔐 安全设计

### Webhook 签名验证

**Alipay:**
- RSA2 (SHA256withRSA) 公钥验证
- 参数排序 + 拼接 + 签名比对
- 实现: `internal/payment/alipay/signature.go`

**Stripe:**
- HMAC-SHA256 webhook secret
- Timestamp 验证防重放攻击
- 由 Stripe SDK 处理

### 幂等性保证

- `payment_transactions.provider_tx_id` 唯一索引
- `GetPaymentTransactionByProviderID()` 检查
- 重复 webhook 返回成功但不创建新 license

## 💾 数据模型

### PaymentTransaction

```sql
CREATE TABLE payment_transactions (
    id SERIAL PRIMARY KEY,
    provider_name VARCHAR(50) NOT NULL,
    provider_tx_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL,
    payment_method VARCHAR(100),
    customer_email VARCHAR(255) NOT NULL,
    license_id INTEGER REFERENCES licenses(id),
    metadata JSONB,
    processed_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(provider_name, provider_tx_id)
);
```

### License (扩展字段)

```sql
ALTER TABLE licenses ADD COLUMN payment_provider VARCHAR(50);
ALTER TABLE licenses ADD COLUMN payment_currency VARCHAR(10);
```

### Plan (扩展字段)

```sql
ALTER TABLE plans ADD COLUMN payment_provider VARCHAR(50);
ALTER TABLE plans ADD COLUMN currency VARCHAR(10) DEFAULT 'USD';
```

## 🌍 Multi-Currency 支持

### 货币配置

```go
CheckoutParams {
    Currency: "CNY",  // Alipay
    Currency: "USD",  // Stripe
}
```

### 金额单位

- **统一:** 所有金额使用最小货币单位 (cents/分)
- **转换:**
  - CNY: 100 分 = 1 元
  - USD: 100 cents = 1 dollar

### Currency 存储

- `payment_transactions.currency` - 实际支付货币
- `licenses.payment_currency` - License 购买货币
- `plans.currency` - Plan 定价货币

## 🔄 支付流程

### 1. Checkout 创建

```
User → Handler.CreateCheckout()
     → Service.CreateCheckout()
     → Provider.CreateCheckout()
     → Return checkout_url
User → Redirect to checkout_url
```

### 2. 支付完成 (Webhook)

```
Provider → POST /api/v1/payment/:provider/webhook
        → Handler.HandleWebhook()
        → Service.HandleWebhook()
        → Provider.VerifyWebhook() ✓
        → Service.handleCompletedPayment()
            ├─ Check idempotency
            ├─ Get plan details
            ├─ Create license
            ├─ Save transaction
            ├─ Send email
            └─ Trigger webhook
```

### 3. 同步回调 (Return URL)

```
Provider → Redirect to /api/v1/payment/:provider/return?session_id=xxx
        → Handler.HandleReturn()
        → Provider.GetPaymentStatus()
        → Return success page with license info
```

## 📊 错误处理

### 分层错误类型

```go
// internal/payment/errors.go
type ErrProviderNotFound struct { Provider string }
type ErrProviderDisabled struct { Provider string }
type ErrPaymentNotFound struct { TransactionID string }
type ErrRefundFailed struct { TransactionID, Reason string }
```

### HTTP 状态码映射

| 错误类型 | HTTP 状态码 |
|---------|------------|
| ErrProviderNotFound | 404 |
| ErrProviderDisabled | 503 |
| ErrPaymentNotFound | 404 |
| Validation Error | 400 |
| Signature Verify Failed | 401 |
| Internal Error | 500 |

## 🧪 测试策略

### 单元测试

- Provider 接口实现
- 签名生成和验证
- License fulfillment 逻辑
- 幂等性检查

### 集成测试

- 端到端 checkout 流程
- Webhook 处理
- 退款流程
- Multi-currency 支持

### Sandbox 环境

**Alipay:**
```
BaseURL: https://openapi-sandbox.dl.alipaydev.com/gateway.do
```

**Stripe:**
```
Use test API keys: sk_test_...
```

## 🚀 扩展新 Provider

### 步骤

1. **创建目录** `internal/payment/<provider>/`

2. **实现接口** `provider.go`
```go
type Provider struct {
    client *Client
    enabled bool
}

func (p *Provider) Name() string { return "<provider>" }
func (p *Provider) CreateCheckout(...) (...) { }
func (p *Provider) VerifyWebhook(...) (...) { }
func (p *Provider) RefundPayment(...) (...) { }
func (p *Provider) GetPaymentStatus(...) (...) { }
```

3. **注册** `register.go`
```go
func init() {
    payment.RegisterProvider("<provider>", NewProvider)
}
```

4. **Import** 在 `cmd/server/main.go`:
```go
import _ "github.com/tabloy/keygate/internal/payment/<provider>"
```

完成！新 provider 自动可用。

## 🎯 最佳实践

### DO ✅

- ✅ 所有金额使用最小货币单位
- ✅ Webhook 必须验证签名
- ✅ 实现幂等性检查
- ✅ 记录所有交易到 payment_transactions
- ✅ Metadata 包含 plan_id 和 product_id
- ✅ 支持 sandbox/test 模式
- ✅ 日志记录关键事件

### DON'T ❌

- ❌ 不要在 handler 层处理业务逻辑
- ❌ 不要跳过签名验证
- ❌ 不要重复创建 license
- ❌ 不要硬编码货币
- ❌ 不要暴露 API secret 到日志
- ❌ 不要依赖同步回调创建 license (必须用 webhook)

## 📖 参考文档

- [Alipay 文档](https://opendocs.alipay.com/open/270/105898)
- [Stripe 文档](https://stripe.com/docs/api)
- [PaymentProvider 接口](./FILES.md)
- [任务列表](./tasks.md)

## 🔮 未来改进

- [ ] 支持订阅自动续费
- [ ] 支付失败重试机制
- [ ] 实时汇率转换
- [ ] 支付分析仪表盘
- [ ] 多步支付流程 (预授权)
- [ ] 货币对冲管理
