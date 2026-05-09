# 迁移任务：gobuffalo/pop → goflower-io/crud

## 数据表结构

> 设计原则：所有字段 NOT NULL；`instance_id` 统一用 `int NOT NULL` 存储雪花 Node ID；`users.id` 由应用层雪花算法生成；其余三张表 `id` 使用数据库 AUTO_INCREMENT。

### 表 1：users（用户表）

```sql
CREATE TABLE `users` (
  `id`                   bigint(20)    NOT NULL,
  `instance_id`          int           NOT NULL DEFAULT 0,
  `aud`                  varchar(255)  NOT NULL DEFAULT '',
  `role`                 varchar(255)  NOT NULL DEFAULT '',
  `email`                varchar(255)  NOT NULL DEFAULT '',
  `encrypted_password`   varchar(255)  NOT NULL DEFAULT '',
  `confirmed_at`         timestamp     NOT NULL DEFAULT '2000-01-01 00:00:00',
  `invited_at`           timestamp     NOT NULL DEFAULT '2000-01-01 00:00:00',
  `confirmation_token`   varchar(255)  NOT NULL DEFAULT '',
  `confirmation_sent_at` timestamp     NOT NULL DEFAULT '2000-01-01 00:00:00',
  `recovery_token`       varchar(255)  NOT NULL DEFAULT '',
  `recovery_sent_at`     timestamp     NOT NULL DEFAULT '2000-01-01 00:00:00',
  `email_change_token`   varchar(255)  NOT NULL DEFAULT '',
  `email_change`         varchar(255)  NOT NULL DEFAULT '',
  `email_change_sent_at` timestamp     NOT NULL DEFAULT '2000-01-01 00:00:00',
  `last_sign_in_at`      timestamp     NOT NULL DEFAULT '2000-01-01 00:00:00',
  `raw_app_meta_data`    varchar(4096) NOT NULL DEFAULT '{}',
  `raw_user_meta_data`   varchar(4096) NOT NULL DEFAULT '{}',
  `is_super_admin`       tinyint(1)    NOT NULL DEFAULT 0,
  `created_at`           timestamp     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`           timestamp     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `users_instance_id_idx` (`instance_id`),
  KEY `users_instance_id_email_idx` (`instance_id`, `email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```

- `id`：bigint，应用层用雪花算法生成，写入前赋值，**不** AUTO_INCREMENT
- `instance_id`：int，存储雪花 Node ID，标识租户；零值 0 表示默认节点
- 时间戳零值用 `2000-01-01 00:00:00`（MySQL 5.7 timestamp 最小有效值为 1970-01-01，`0001-01-01` 越界）

---

### 表 2：instances（实例/租户表）

```sql
CREATE TABLE `instances` (
  `id`              bigint(20)   NOT NULL AUTO_INCREMENT,
  `uuid`            varchar(255) NOT NULL DEFAULT '',
  `raw_base_config` text         NOT NULL DEFAULT '',
  `created_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `instances_uuid_uidx` (`uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```

---

### 表 3：refresh_tokens（刷新令牌表）

```sql
CREATE TABLE `refresh_tokens` (
  `id`          bigint(20)   NOT NULL AUTO_INCREMENT,
  `instance_id` int          NOT NULL DEFAULT 0,
  `token`       varchar(255) NOT NULL DEFAULT '',
  `user_id`     bigint(20)   NOT NULL DEFAULT 0,
  `revoked`     tinyint(1)   NOT NULL DEFAULT 0,
  `created_at`  timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `refresh_tokens_instance_id_idx` (`instance_id`),
  KEY `refresh_tokens_instance_id_user_id_idx` (`instance_id`, `user_id`),
  KEY `refresh_tokens_token_idx` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```

- `user_id`：bigint，对应 `users.id`（雪花 ID）

---

### 表 4：audit_log_entries（审计日志表）

```sql
CREATE TABLE `audit_log_entries` (
  `id`          bigint(20)   NOT NULL AUTO_INCREMENT,
  `instance_id` int          NOT NULL DEFAULT 0,
  `payload`     varchar(4096) NOT NULL DEFAULT '{}',
  `created_at`  timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `audit_logs_instance_id_idx` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```

---

## 迁移挑战

| 挑战 | 原始方案 | 新方案 |
|------|---------|--------|
| users.id 类型变更 | varchar(255) UUID 主键 | bigint 雪花 ID，应用层生成；引入 `github.com/bwmarrin/snowflake` |
| instance_id 类型变更 | varchar(255) UUID | int，存雪花 Node ID（0～1023）；全表统一 |
| refresh_tokens.user_id 类型变更 | varchar(255) UUID | bigint，与 users.id 对齐 |
| audit_log_entries.id 类型变更 | varchar(255) UUID | bigint AUTO_INCREMENT |
| instances.id 类型变更 | varchar(255) UUID | bigint AUTO_INCREMENT |
| timestamp NOT NULL | 原始全为 NULL | 统一 NOT NULL，零值用 `2000-01-01 00:00:00`；Go 侧用 `time.Time`（非指针）|
| JSON 字段 | json 类型（可 NULL）| varchar(4096) NOT NULL DEFAULT '{}'；保留 json_map.go Scanner/Valuer |
| 命名空间（多租户）| TableName() 动态 `{ns}_users` | goflower-io/crud 固定表名；在连接层处理 ns |
| 生命周期钩子 | pop BeforeCreate/BeforeUpdate | Create 前手动调用 validate()，再赋雪花 ID |
| UpdateOnly | `tx.UpdateOnly(model, fields...)` | crud `Update().SetXxx().Where()` 模式 |
| 原始 SQL | `tx.RawQuery().Exec()` | 通过底层 `*sql.DB` 直接执行 |
| 迁移系统 | pop FileMigrator | golang-migrate/migrate 或直接执行 SQL |
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
- [ ] 运行以下命令重新生成 `crud/` 目录下 Go 模型代码：
  ```
  crud -path crud/sql
  ```
- [ ] 检查生成的各模型文件

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
- [ ] 引入雪花 ID 生成器（`github.com/bwmarrin/snowflake`），初始化全局 `snowflake.Node`
- [ ] `(u *User) Create()` — 在 validate 后、写库前调用 `node.Generate().Int64()` 为 `u.ID` 赋值，再调用 `client.User.Create().SetUser(u).Save(ctx)`

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
