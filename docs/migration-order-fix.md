# 数据库迁移顺序修复

## 问题

数据库迁移失败，错误信息：
```
ERROR: relation "renewal_reminders" does not exist
```

## 原因

迁移文件按字母顺序执行，但 `20261009_additional_indexes.up.sql` 在字母顺序上排在 `20261009_renewal_reminders.up.sql` 之前，导致尝试在尚未创建的表上创建索引。

## 解决方案

将 `additional_indexes` 迁移文件重命名为 `z_additional_indexes`，确保它在所有其他同日期的迁移之后执行。

### 重命名的文件

```
20261009_additional_indexes.up.sql   → 20261009_z_additional_indexes.up.sql
20261009_additional_indexes.down.sql → 20261009_z_additional_indexes.down.sql
```

## 正确的执行顺序

现在迁移按以下顺序执行：

1. ✅ **20261009_licenses_payment_fields** - 添加payment相关字段到licenses表
2. ✅ **20261009_payment_transactions** - 创建payment_transactions表
3. ✅ **20261009_plans_multi_currency** - 添加多币种字段到plans表
4. ✅ **20261009_renewal_reminders** - 创建renewal_reminders表
5. ✅ **20261009_z_additional_indexes** - 在已存在的表上创建性能优化索引

## 索引详情

`20261009_z_additional_indexes.up.sql` 创建以下索引：

1. **idx_licenses_valid_until** - 在 `licenses(valid_until)` 上
   - 用途：加速续费提醒查询
   
2. **idx_renewal_reminders_license_type** - 在 `renewal_reminders(license_id, reminder_type)` 上
   - 用途：快速查找特定许可证的特定类型提醒

## 验证

重新运行数据库迁移：

```bash
docker compose down
docker compose up -d postgres
docker compose up keygate
```

应该看到：
```
✓ Applied migration: 20261009_licenses_payment_fields
✓ Applied migration: 20261009_payment_transactions
✓ Applied migration: 20261009_plans_multi_currency
✓ Applied migration: 20261009_renewal_reminders
✓ Applied migration: 20261009_z_additional_indexes
```

## 回滚顺序

如果需要回滚，down迁移按相反顺序执行：

1. 20261009_z_additional_indexes.down.sql
2. 20261009_renewal_reminders.down.sql
3. 20261009_plans_multi_currency.down.sql
4. 20261009_payment_transactions.down.sql
5. 20261009_licenses_payment_fields.down.sql

## 注意事项

- 迁移文件名使用字母顺序排序
- 同一日期的迁移需要特别注意依赖关系
- 使用 `z_` 前缀可以确保文件在同批次中最后执行
- 避免在同一天创建多个相互依赖的迁移，或使用更精确的时间戳

## 相关文档

- [数据库迁移文档](./database-migrations.md)
- [多支付网关集成](../openspec/changes/multi-payment-gateway-integration/)
