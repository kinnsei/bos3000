# TOTP 二次验证 (2FA) 实施计划

## 概述

为 BOS3000 管理后台和客户门户增加基于 TOTP (Time-based One-Time Password) 的二次验证，提升账户安全性。管理员账号强制开启，客户账号可选开启。

## 技术选型

| 组件 | 方案 |
|------|------|
| TOTP 库 | `github.com/pquerna/otp` (Go, 成熟稳定) |
| QR 码生成 | `github.com/skip2/go-qrcode` |
| 恢复码 | 8 位随机字母数字, 生成 10 个 |
| 前端 OTP 输入 | 6 位数字输入框组件 (shadcn + custom) |

## 数据库设计

### Migration: `auth/migrations/X_add_totp.up.sql`

```sql
ALTER TABLE users ADD COLUMN totp_secret TEXT;
ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN totp_verified_at TIMESTAMPTZ;

CREATE TABLE totp_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_recovery_codes_user ON totp_recovery_codes(user_id);
```

## API 端点设计

### Phase 1: 管理后台 (Admin)

| 方法 | 路径 | 描述 | 权限 |
|------|------|------|------|
| POST | `/auth/totp/setup` | 生成 TOTP secret + QR URI | auth |
| POST | `/auth/totp/verify-setup` | 验证首次 TOTP 码并激活 | auth |
| POST | `/auth/totp/validate` | 登录时验证 TOTP 码 | public |
| POST | `/auth/totp/disable` | 禁用 2FA (需当前 TOTP 码) | auth |
| GET  | `/auth/totp/recovery-codes` | 获取恢复码 | auth |
| POST | `/auth/totp/recovery-codes/regenerate` | 重新生成恢复码 | auth |

### Phase 2: 客户门户 (Portal)

复用同一套后端端点，通过 auth middleware 区分 admin/client 角色。

## 登录流程变更

```
当前:  email+password → JWT → 进入系统
新流程: email+password → { requires_2fa: true, temp_token } → TOTP 验证页 → JWT → 进入系统
```

### 详细流程:

1. **POST /auth/admin/login** (或 client/login)
   - 密码验证通过后，检查 `totp_enabled`
   - 如果未开启 2FA → 直接返回 JWT (现有行为)
   - 如果已开启 2FA → 返回 `{ requires_2fa: true, temp_token: "xxx" }`
   - `temp_token` 是短期 JWT (5分钟有效), claims 包含 `user_id` + `pending_2fa: true`

2. **POST /auth/totp/validate**
   - 接收 `{ temp_token, code }`
   - 验证 TOTP 码 (支持 ±1 时间窗口)
   - 如果是恢复码 → 标记已用
   - 验证通过 → 返回正式 JWT

## 前端实现

### 新增组件

1. **OTP 输入组件** — `components/shared/otp-input.tsx`
   - 6 个独立数字输入框, 自动跳转
   - 粘贴支持, 退格删除
   - 恢复码切换链接

2. **TOTP 验证页** — `features/auth/totp-verify.tsx`
   - 登录后跳转, 输入 6 位 TOTP 码
   - "使用恢复码" 切换选项
   - 倒计时提示 temp_token 过期

3. **2FA 设置页** — `features/settings/totp-setup.tsx`
   - QR 码显示 (用于 Google Authenticator / Authy 扫码)
   - 手动输入 secret key 选项
   - 首次验证确认
   - 恢复码展示 + 下载/复制

4. **路由变更**
   - 新增 `/totp-verify` 路由 (不需要 AuthGuard)
   - 设置页增加 "安全设置" 区块

### 登录流程前端逻辑变更

```typescript
// login.tsx handleSubmit 变更
const result = await login.mutateAsync({ email, password })
if (result.requires_2fa) {
  // 保存 temp_token, 跳转验证页
  sessionStorage.setItem('temp_token', result.temp_token)
  navigate('/totp-verify')
} else {
  navigate('/', { replace: true })
}
```

## 实施阶段

### Phase 1: 后端核心 (预计 1-2 天)
- [ ] 数据库 migration
- [ ] TOTP secret 生成、验证工具函数
- [ ] Setup / verify-setup / validate / disable 端点
- [ ] 恢复码生成与验证
- [ ] 修改 login 端点支持 2FA 流程
- [ ] 单元测试

### Phase 2: Admin 前端 (预计 1 天)
- [ ] OTP 输入组件
- [ ] TOTP 验证页
- [ ] 2FA 设置页 (QR 码 + 恢复码)
- [ ] 路由与登录流程集成
- [ ] E2E 测试

### Phase 3: Portal 前端 (预计 0.5 天)
- [ ] 复用 Admin 组件到 Portal
- [ ] 客户设置页增加 2FA 开关
- [ ] 测试

### Phase 4: 策略与强制 (预计 0.5 天)
- [ ] 管理员强制开启逻辑 (首次登录引导设置)
- [ ] 管理员可强制客户开启 2FA
- [ ] 2FA 状态在客户管理列表显示

## 安全注意事项

- TOTP secret 加密存储 (AES-256-GCM, key 从 Encore secrets 获取)
- Rate limit: TOTP 验证 5 次/5分钟
- temp_token 严格 5 分钟过期
- 恢复码一次性使用, bcrypt 哈希存储
- 禁用 2FA 必须提供当前有效 TOTP 码
- 审计日志记录所有 2FA 相关操作
