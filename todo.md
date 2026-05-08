# 迁移任务：gobuffalo/pop → goflower-io/crud

## 数据表结构

### 表 1：users（用户表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint(20) NOT NULL AUTO_INCREMENT PK | 自增主键 |
| uid | varchar(255) NOT NULL DEFAULT '' | UUID，UNIQUE KEY |
| instance_id | varchar(255) NOT NULL DEFAULT '' | 租户 ID，索引 |
| aud | varchar(255) NOT NULL DEFAULT '' | audience |
| role | varchar(255) NOT NULL DEFAULT '' | 角色 |
| email | varchar(255) NOT NULL DEFAULT '' | 邮箱，联合索引 (instance_id, email) |
| encrypted_password | varchar(255) NOT NULL DEFAULT '' | 加密密码（bcrypt）|
| confirmed_at | timestamp NOT NULL DEFAULT '0001-01-01 00:00:00' | 邮箱确认时间，零值表示未确认 |
| invited_at | timestamp NOT NULL DEFAULT '0001-01-01 00:00:00' | 邀请时间，零值表示未邀请 |
| confirmation_token | varchar(255) NOT NULL DEFAULT '' | 确认令牌 |
| confirmation_sent_at | timestamp NOT NULL DEFAULT '0001-01-01 00:00:00' | 确认邮件发送时间 |
| recovery_token | varchar(255) NOT NULL DEFAULT '' | 恢复令牌 |
| recovery_sent_at | timestamp NOT NULL DEFAULT '0001-01-01 00:00:00' | 恢复邮件发送时间 |
| email_change_token | varchar(255) NOT NULL DEFAULT '' | 改邮箱令牌 |
| email_change | varchar(255) NOT NULL DEFAULT '' | 新邮箱地址 |
| email_change_sent_at | timestamp NOT NULL DEFAULT '0001-01-01 00:00:00' | 改邮箱邮件发送时间 |
| last_sign_in_at | timestamp NOT NULL DEFAULT '0001-01-01 00:00:00' | 最后登录时间，零值表示从未登录 |
| raw_app_meta_data | varchar(4096) NOT NULL DEFAULT '{}' | 应用元数据（JSON 字符串）|
| raw_user_meta_data | varchar(4096) NOT NULL DEFAULT '{}' | 用户元数据（JSON 字符串）|
| is_super_admin | tinyint(1) NOT NULL DEFAULT 0 | 超级管理员标志 |
| created_at | timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP | 更新时间 |

索引：`users_uid_uidx(uid)` UNIQUE、`users_instance_id_idx(instance_id)`、`users_instance_id_email_idx(instance_id, email)`

---

### 表 2：instances（实例/租户表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint(20) NOT NULL AUTO_INCREMENT PK | 自增主键 |
| uid | varchar(255) NOT NULL DEFAULT '' | UUID，UNIQUE KEY |
| uuid | varchar(255) NOT NULL DEFAULT '' | 对外 UUID，UNIQUE KEY |
| raw_base_config | text NOT NULL DEFAULT '' | JSON 格式的租户配置 |
| created_at | timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP | 更新时间 |

---

### 表 3：refresh_tokens（刷新令牌表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint(20) NOT NULL AUTO_INCREMENT PK | 自增主键 |
| instance_id | varchar(255) NOT NULL DEFAULT '' | 租户 ID，索引 |
| token | varchar(255) NOT NULL DEFAULT '' | 令牌字符串，索引 |
| user_id | varchar(255) NOT NULL DEFAULT '' | 用户 UUID，联合索引 (instance_id, user_id) |
| revoked | tinyint(1) NOT NULL DEFAULT 0 | 是否已撤销，0=有效 1=已撤销 |
| created_at | timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP | 更新时间 |

索引：`refresh_tokens_instance_id_idx(instance_id)`、`refresh_tokens_instance_id_user_id_idx(instance_id, user_id)`、`refresh_tokens_token_idx(token)`

---

### 表 4：audit_log_entries（审计日志表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint(20) NOT NULL AUTO_INCREMENT PK | 自增主键 |
| uid | varchar(255) NOT NULL DEFAULT '' | UUID，UNIQUE KEY |
| instance_id | varchar(255) NOT NULL DEFAULT '' | 租户 ID，索引 |
| payload | varchar(4096) NOT NULL DEFAULT '{}' | 审计数据 JSON（actor_id/email、action、log_type 等）|
| created_at | timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP | 创建时间 |

索引：`audit_log_entries_uid_uidx(uid)` UNIQUE、`audit_logs_instance_id_idx(instance_id)`

---

## 迁移挑战

| 挑战 | 当前方案 | 迁移方案 |
|------|---------|---------|
| UUID 主键 | pop 原生支持 uuid.UUID | 用 string 类型，手动处理 UUID 生成 |
| 命名空间（多租户）| TableName() 动态返回 `{ns}_users` | goflower-io/crud 生成固定表名；需在连接层处理 ns |
| 生命周期钩子 | pop BeforeCreate/BeforeUpdate | 在 Save/Create 前手动调用验证逻辑 |
| JSON 字段 | pop 的 JSONMap 类型 | 保留 json_map.go，实现 database/sql Scanner/Valuer 接口 |
| NULL 时间戳 | `*time.Time` | 保留 `*time.Time`，通过自定义 Scanner |
| 迁移系统 | pop FileMigrator | 改用 golang-migrate/migrate 或直接执行 SQL |
| UpdateOnly | `tx.UpdateOnly(model, fields...)` | crud 的 `Update().SetXxx().Where()` 模式 |
| 原始 SQL | `tx.RawQuery().Exec()` | 通过底层 `*sql.DB` 直接执行 |
| 分页总数 | `q.Paginator.TotalEntriesSize` | 额外执行 COUNT 查询 |

---

## 迁移任务清单

### Phase 0：准备工作
- [ ] 安装 goflower-io/crud CLI：`go install github.com/goflower-io/crud@main`
- [ ] 添加依赖：`go get github.com/goflower-io/crud`
- [ ] 创建 `crud/sql/` 目录结构

### Phase 1：编写 SQL DDL（crud/sql/）
- [ ] `crud/sql/users.sql` — users 表 DDL（适配 crud 工具格式）
- [ ] `crud/sql/instances.sql` — instances 表 DDL
- [ ] `crud/sql/refresh_tokens.sql` — refresh_tokens 表 DDL
- [ ] `crud/sql/audit_log_entries.sql` — audit_log_entries 表 DDL

### Phase 2：生成模型代码
- [ ] 运行 `crud` 命令，生成 `crud/` 目录下 Go 模型代码
- [ ] 检查生成的 `crud/aa_client.go`（客户端）和各模型文件

### Phase 3：重写 storage 层（storage/dial.go）
- [ ] 用 `crud.NewClient()` 替换 `pop.NewConnection()`
- [ ] 重定义 `Connection` struct，包装 crud Client
- [ ] 重写 `Transaction()` 方法（改用 `client.Begin(ctx)`）
- [ ] 重写 `UpdateOnly()` 方法（改为 crud Update API）

### Phase 4：逐表迁移 models/

#### models/user.go（最复杂，411 行，8 个查询方法 + 8 个更新方法）
- [ ] `findUser()` — 通用查询辅助函数
- [ ] `FindUserByID()` — `client.User.Find().Where(IdOp.EQ(id)).One(ctx)`
- [ ] `FindUserByEmailAndAudience()` — Where 多条件查询
- [ ] `FindUserByInstanceIDAndID()` — Where 多条件查询
- [ ] `FindUserByConfirmationToken()` — Where token 查询
- [ ] `FindUserByRecoveryToken()` — Where token 查询
- [ ] `FindUserWithRefreshToken()` — 联合查询 refresh_tokens + users
- [ ] `FindUsersInAudience()` — 分页 + 排序 + 模糊过滤（最复杂）
- [ ] `FindUsersForExportInAudience()` — 无分页的全量查询
- [ ] `CountOtherUsers()` — Count 查询
- [ ] `IsDuplicatedEmail()` — Count 查询判断
- [ ] `(u *User) SetRole()` — 更新 role 字段
- [ ] `(u *User) Confirm()` — 更新 confirmation_token + confirmed_at
- [ ] `(u *User) UpdateUserMetaData()` — 更新 raw_user_meta_data
- [ ] `(u *User) UpdateAppMetaData()` — 更新 raw_app_meta_data
- [ ] `(u *User) SetEmail()` — 更新 email
- [ ] `(u *User) UpdatePassword()` — 更新 encrypted_password
- [ ] `(u *User) ConfirmEmailChange()` — 更新 email + email_change + email_change_token
- [ ] `(u *User) Recover()` — 更新 recovery_token
- [ ] 替换 `BeforeCreate/BeforeUpdate` 钩子为手动调用的 `validate()` 方法
- [ ] `(u *User) Create()` — 调用 validate + `client.User.Create().SetUser(u).Save(ctx)`

#### models/instance.go
- [ ] `GetInstance()` — `client.Instance.Find().Where(IdOp.EQ(id)).One(ctx)`
- [ ] `GetInstanceByUUID()` — Where uuid 查询
- [ ] `(i *Instance) UpdateConfig()` — 更新 raw_base_config
- [ ] `DeleteInstance()` — 事务：删除关联 users/refresh_tokens，再删 instance

#### models/refresh_token.go
- [ ] `createRefreshToken()` — `client.RefreshToken.Create().Set(...).Save(ctx)`
- [ ] `GrantAuthenticatedUser()` — 创建新 refresh_token
- [ ] `GrantRefreshTokenSwap()` — 事务：撤销旧 token + 创建新 token
- [ ] `Logout()` — `DELETE FROM refresh_tokens WHERE instance_id=? AND user_id=?`

#### models/audit_log_entry.go
- [ ] `NewAuditLogEntry()` — `client.AuditLogEntry.Create().Set(...).Save(ctx)`
- [ ] `FindAuditLogEntries()` — Where + Order + Paginate 查询

#### models/connection.go
- [ ] `TruncateAll()` — 通过底层 `*sql.DB` 执行 TRUNCATE

### Phase 5：处理特殊问题
- [ ] **命名空间**：评估方案（动态表名 vs 应用层隔离），选择并实现
- [ ] **JSON 字段**：验证 `models/json_map.go` 的 `Scan()`/`Value()` 与新 ORM 兼容
- [ ] **NULL 时间戳**：统一使用 `*time.Time` + 自定义 Scanner 处理
- [ ] **错误映射**：将 `sql.ErrNoRows` 映射为 `UserNotFoundError{}`、`InstanceNotFoundError{}` 等

### Phase 6：替换迁移命令（cmd/migrate_cmd.go）
- [ ] 选择方案：
  - 方案 A：引入 `golang-migrate/migrate` 替换 pop FileMigrator
  - 方案 B：直接读取 `migrations/*.sql` 文件，用 `*sql.DB` 执行
- [ ] 实现 migrate up / migrate status 命令

### Phase 7：清理依赖
- [ ] `tools.go` — 删除 `gobuffalo/pop/v5/soda` 工具引用
- [ ] 运行 `go mod tidy`，移除所有 gobuffalo/pop 相关依赖
- [ ] 检查 `go.sum` 确认清理完成

### Phase 8：测试验证
- [ ] `go build ./...` — 确保编译通过
- [ ] 更新 `storage/test/db_setup.go` 以适配新的 crud Client
- [ ] 运行 `go test ./models/...`
- [ ] 运行 `go test ./api/...`
- [ ] 手动测试关键流程：注册、登录、刷新令牌、登出

---

## 关键文件速查

```
storage/dial.go              # 数据库连接层（核心改动）
models/user.go               # 用户模型（最复杂）
models/refresh_token.go      # 刷新令牌模型
models/instance.go           # 实例模型
models/audit_log_entry.go    # 审计日志模型
models/connection.go         # 工具方法
models/json_map.go           # JSON 类型（保留）
cmd/migrate_cmd.go           # 迁移命令
tools.go                     # 工具依赖
go.mod                       # 依赖管理
migrations/schema.sql        # 完整 DDL 参考
```
