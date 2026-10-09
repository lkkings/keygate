# 更新Lockfile说明

## 问题

Docker构建失败，因为添加了新的依赖但lockfile没有更新：

```
error: lockfile had changes, but lockfile is frozen
```

## 新增的依赖

为了支持支付配置UI，添加了两个Radix UI组件：

- `@radix-ui/react-checkbox` - 用于RefundModal和支付设置
- `@radix-ui/react-radio-group` - 用于PaymentMethodSelector和RenewalPage

## 解决方案

### 方法1: 使用提供的脚本（推荐）

**Windows用户:**
```bash
update-lockfile.bat
```

**Linux/Mac用户:**
```bash
./update-lockfile.sh
```

### 方法2: 手动更新

```bash
cd web
bun install
```

这将：
1. 安装新依赖
2. 更新 `bun.lockb` 文件
3. 确保lockfile与package.json同步

### 方法3: 使用npm（如果bun不可用）

```bash
cd web
npm install
# 然后将生成的package-lock.json提交
```

## 验证

更新lockfile后，验证Docker构建：

```bash
docker compose build keygate
```

应该看到：
```
✓ 2488 modules transformed
✓ built in XX.XXs
```

## 受影响的组件

新UI组件已创建在 `web/src/components/ui/`:

1. **checkbox.tsx** - 复选框组件
   - 用于: RefundModal（撤销许可证选项）
   - 用于: 支付设置（启用/禁用功能）

2. **radio-group.tsx** - 单选按钮组
   - 用于: PaymentMethodSelector（选择支付方式）
   - 用于: RenewalPage（选择续费计划）

3. **switch.tsx** - 开关组件
   - 用于: 支付提供商启用/禁用
   - 用于: 沙箱模式开关

4. **textarea.tsx** - 多行文本输入
   - 用于: API密钥输入（Alipay私钥等）

## 构建状态

✅ **TypeScript编译**: 成功
✅ **Vite构建**: 成功  
⏳ **Docker构建**: 需要更新lockfile

## 下一步

1. 运行 `update-lockfile.bat` 或 `bun install`
2. 提交更新的 `web/bun.lockb`
3. 重新构建Docker镜像
4. 启动应用并访问支付配置

---

**注意**: 如果您使用的是npm而不是bun，请确保：
- 删除 `web/bun.lockb`（如果存在）
- 使用 `npm install` 生成 `package-lock.json`
- 更新Dockerfile使用npm而不是bun
