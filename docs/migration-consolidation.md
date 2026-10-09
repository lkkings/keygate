# 数据库迁移合并说明

## 概述

已将所有39个增量迁移文件合并为单个初始化脚本，用于全新安装。

## 为什么要合并？

### 原来的问题
- 39个独立的迁移文件
- 文件之间存在依赖关系（例如：additional_indexes依赖renewal_reminders）
- 首次安装需要按顺序执行所有迁移
- 迁移顺序错误导致失败

### 合并后的优势
- ✅ **单文件初始化** - 一次性创建所有表
- ✅ **无依赖问题** - 按正确顺序定义所有对象
- ✅ **更快启动** - 不需要逐个应用迁移
- ✅ **更清晰** - 完整的数据库架构一目了然
- ✅ **适合全新安装** - 您表示没有需要迁移的现有数据

## 新的迁移文件

### 主文件
```
db/migrations/00000000_init.up.sql    - 完整的数据库架构初始化
db/migrations/00000000_init.down.sql  - 清理脚本（删除所有表）
```

### 包含的表（按类别）

#### 核心表
- `users` - 用户账户
- `products` - 产品
- `plans` - 定价计划（支持多币种：USD, CNY, HKD）
- `licenses` - 许可证密钥
- `activations` - 设备激活

#### 支付表
- `payment_transactions` - 支付交易记录

#### 通知和邮件
- `email_queue` - 邮件队列
- `renewal_reminders` - 续费提醒追踪
- `notifications` - 应用内通知

#### 功能扩展
- `license_addons` - 许可证附加组件

#### 发布管理
- `releases` - 软件发布
- `release_artifacts` - 下载文件
- `release_signing_keys` - 签名密钥

#### 认证和安全
- `refresh_tokens` - JWT刷新令牌
- `otp_codes` - 一次性密码

#### 系统表
- `settings` - 全局设置
- `processed_events` - 事件去重追踪

## 使用方法

### 方法1：清理旧迁移（推荐）

运行清理脚本：
```bash
cleanup-old-migrations.bat
```

这将：
1. 将所有 `202*.sql` 文件移动到 `db/migrations.backup/`
2. 保留 `00000000_init.*.sql` 文件
3. 允许全新的数据库初始化

### 方法2：手动清理

```bash
# 创建备份目录
mkdir db\migrations.backup

# 移动旧迁移文件
move db\migrations\202*.sql db\migrations.backup\

# 验证只剩新文件
dir db\migrations\*.sql
```

应该只看到：
- `00000000_init.up.sql`
- `00000000_init.down.sql`

## 启动全新数据库

```bash
# 1. 停止并删除旧数据库
docker compose down -v

# 2. 启动新数据库（将自动应用初始化脚本）
docker compose up
```

## 迁移系统的工作原理

KeyGate的迁移系统：
1. 读取 `db/migrations/` 目录下的所有 `.up.sql` 文件
2. 按**字母顺序**排序执行
3. 使用 `schema_migrations` 表追踪已应用的迁移
4. `00000000_` 前缀确保初始化脚本最先执行

## 架构特性

### 索引优化
- 邮箱查询使用不区分大小写索引：`LOWER(email)`
- 许可证过期查询：`valid_until` 部分索引
- 支付交易查询：复合索引 `(provider_name, status)`
- 续费提醒：复合索引 `(license_id, reminder_type)`

### 约束
- 外键级联删除保证数据一致性
- CHECK约束验证枚举值
- UNIQUE约束防止重复

### 多币种支持
Plans表支持三种货币：
- `price_usd` - 美元（分）
- `price_cny` - 人民币（分）
- `price_hkd` - 港币（分）

### 支付集成
- `payment_provider` - 支持多个支付提供商
- `payment_transaction_id` - 外部交易ID
- `refund_amount` - 退款金额追踪

## 备份的旧迁移文件

如果需要查看历史迁移记录，所有原始文件都保存在 `db/migrations.backup/` 中：
- 39个 `.up.sql` 文件
- 39个 `.down.sql` 文件

## 故障排除

### 问题：迁移表已存在

如果看到"schema_migrations already exists"错误：

```bash
# 完全重置数据库
docker compose down -v
docker compose up
```

### 问题：仍然报错"relation does not exist"

确保：
1. 已运行 `cleanup-old-migrations.bat`
2. 只有 `00000000_init.*.sql` 在 `db/migrations/` 目录
3. 运行 `docker compose down -v` 删除了旧数据

### 问题：需要恢复旧迁移

```bash
# 从备份恢复
copy db\migrations.backup\*.sql db\migrations\
del db\migrations\00000000_init.*.sql
```

## 与现有系统的兼容性

### 全新安装 ✅
完美支持！这就是设计目的。

### 已有数据的升级 ⚠️
如果您有现有数据库：
1. **不要**清理旧迁移文件
2. 保持增量迁移
3. 或者先导出数据，使用新架构，再导入

## 下一步

1. ✅ 运行 `cleanup-old-migrations.bat`
2. ✅ 运行 `update-lockfile.bat` 更新前端依赖
3. ✅ 运行 `docker compose down -v && docker compose up`
4. ✅ 访问管理后台配置支付提供商

## 相关文档

- [迁移顺序修复](./migration-order-fix.md)
- [Lockfile更新说明](./lockfile-update.md)
- [TypeScript错误修复](./typescript-fixes.md)
