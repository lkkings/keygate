# 多支付网关集成 - 关键文件清单

## 📁 核心架构层

### Payment Core (`internal/payment/`)
- **provider.go** - PaymentProvider 接口定义
- **types.go** - 通用数据类型 (CheckoutParams, WebhookEvent, etc.)
- **registry.go** - Provider 注册表和工厂模式
- **errors.go** - 标准化错误类型

## 📁 Alipay 实现 (`internal/payment/alipay/`)
- **client.go** - Alipay API 客户端
- **signature.go** - RSA2 签名生成和验证
- **encoding.go** - URL 编码工具
- **provider.go** - Alipay PaymentProvider 实现
- **register.go** - 自动注册到 registry

## 📁 Stripe 实现 (`internal/payment/stripe/`)
- **adapter.go** - Stripe PaymentProvider 实现 (完全重构)
- **register.go** - 从环境变量加载配置

## 📁 服务层 (`internal/service/`)
- **payment.go** - PaymentService 业务逻辑层
  - 事务管理
  - Provider 路由
  - **License fulfillment 完整实现**
  - Email 和 Webhook 触发

## 📁 Handler 层 (`internal/handler/payment/`)
- **handler.go** - HTTP 路由处理器
  - POST /api/v1/payment/:provider/checkout
  - POST /api/v1/payment/:provider/webhook
  - GET /api/v1/payment/:provider/return
  - GET /api/v1/payment/:provider/status/:transaction_id
  - GET /api/v1/payment/providers
  - POST /api/v1/admin/payment/:provider/refund/:transaction_id

## 📁 数据存储层 (`internal/store/`)
- **payment_transaction.go** - 支付事务 CRUD

## 📁 数据模型 (`internal/model/`)
- **payment_transaction.go** - PaymentTransaction 结构体

## 📁 数据库 (`db/migrations/`)
- **XXXXXX_payment_gateway_support.sql** - payment_transactions 表
- **XXXXXX_add_payment_provider_to_plans.sql** - plans 表扩展
- **XXXXXX_add_payment_currency.sql** - licenses 表货币字段
- **XXXXXX_add_checkout_id_to_plans.sql** - plans 表统一 checkout
- **XXXXXX_drop_payment_tables.sql** - 清理旧表

## 📁 主程序 (`cmd/server/`)
- **main.go** - 路由注册和依赖注入

## 🔗 依赖关系

```
main.go
  ↓
handler/payment/handler.go
  ↓
service/payment.go
  ↓
payment/registry.go → payment/provider.go
  ↓                         ↓
  ├─ alipay/provider.go ←─ alipay/client.go
  │                        alipay/signature.go
  │
  └─ stripe/adapter.go
  ↓
store/payment_transaction.go
  ↓
model/payment_transaction.go
```

## 📊 统计

### 代码行数 (估计)
- **核心接口和类型:** ~500 行
- **Alipay 实现:** ~1,200 行
- **Stripe 实现:** ~250 行  
- **服务层 (含 fulfillment):** ~480 行
- **Handler 层:** ~450 行
- **Store 层:** ~250 行
- **Migration SQL:** ~100 行

**总计:** 约 3,230 行代码

### 已完成任务
- **37/139 任务** (~26.6%)
- **9 个任务组**完全完成

### 文件清单
- **25 个新文件**创建
- **1 个文件**重构 (main.go)

## 🎯 下一步优先级

1. **续费逻辑** - 扩展现有 license 而非创建新的
2. **测试覆盖** - Alipay 和 Stripe 的单元/集成测试
3. **Admin Settings API** - 支付网关配置管理
4. **文档编写** - API 文档和开发者指南
5. **前端集成** - Payment provider 选择 UI

## 🛡️ 关键特性

### ✅ 已实现
- ✅ Provider 抽象层和工厂模式
- ✅ 完整的 Alipay 集成 (签名、支付、验证)
- ✅ 重构的 Stripe 集成 (完全独立)
- ✅ 统一的 Webhook 处理
- ✅ 事务持久化
- ✅ Multi-currency 支持
- ✅ 幂等性处理 (从 middleware 继承)
- ✅ 错误类型化和传播

### ⏳ 待实现
- ⏳ License fulfillment 自动化
- ⏳ 退款流程
- ⏳ 支付状态查询
- ⏳ Admin 配置界面
- ⏳ 测试覆盖
- ⏳ 前端集成

## 📖 架构亮点

### 1. **接口驱动设计**
所有 provider 实现同一个 `PaymentProvider` 接口，易于扩展。

### 2. **工厂模式注册**
Provider 通过 `init()` 自动注册，main.go 零配置。

### 3. **分层清晰**
Handler → Service → Provider → Store，职责明确。

### 4. **错误处理统一**
自定义错误类型 + HTTP 状态码映射。

### 5. **配置灵活**
支持环境变量、数据库存储、运行时配置。

### 6. **Webhook 安全**
- Alipay: RSA2 签名验证
- Stripe: Webhook secret HMAC

### 7. **事务管理**
所有支付事件持久化到 `payment_transactions` 表。

### 8. **货币支持**
- Alipay: CNY
- Stripe: USD, EUR, GBP, 等多种货币
