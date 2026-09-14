# 岗位实训系统 · 部署指南

本指南覆盖四种部署路径，按推荐顺序排列：

| 方式 | 适用场景 | 数据库 |
|---|---|---|
| [方式一 · Docker Compose](#方式一docker-compose推荐) | **推荐**。一条命令拉起数据库 + 后端 | PostgreSQL |
| [方式二 · Docker 单容器](#方式二docker-单容器) | 已有托管数据库，只需跑后端 | PostgreSQL |
| [方式三 · 直接运行二进制](#方式三直接运行二进制) | 无 Docker 环境 / 内网离线 | PostgreSQL 或 SQLite |
| [方式四 · SQLite 快速演示](#方式四sqlite-快速演示) | 本地试用、演示，不需要 PostgreSQL | SQLite |

> **镜像来源**：CI（GitHub Actions）已构建并推送镜像
> `ghcr.io/ziruibai/training-system:latest`，下文所有 Docker 方式均可直接使用该镜像，
> 无需本地构建。

---

## 目录

- [0. 部署包内容](#0-部署包内容)
- [1. 环境变量说明](#1-环境变量说明)
- [2. 方式一 · Docker Compose（推荐）](#方式一docker-compose推荐)
- [3. 方式二 · Docker 单容器](#方式二docker-单容器)
- [4. 方式三 · 直接运行二进制](#方式三直接运行二进制)
- [5. 方式四 · SQLite 快速演示](#方式四sqlite-快速演示)
- [6. 数据库初始化说明](#6-数据库初始化说明)
- [7. 如何检查服务是否启动成功](#7-如何检查服务是否启动成功)
- [8. 演示账号](#8-演示账号)
- [9. 常见问题](#9-常见问题)
- [10. 生产环境注意事项](#10-生产环境注意事项)

---

## 0. 部署包内容

```
server                                   # 生产单文件二进制（linux/amd64，前端已内嵌，无需 Node）
backend/                                 # Go 后端源码（cmd/ + internal/ + go.mod/go.sum）
backend/Dockerfile                       # 多阶段镜像构建文件（已内嵌 goproxy.cn 镜像源配置）
backend/migrations/postgres/001_init.sql # PostgreSQL 建表 SQL（基础 16 张表）
backend/migrations/postgres/002_*.sql    # PostgreSQL 建表 SQL（补丁，模板 + 辅导 3 张表）
backend/var/training.db                  # SQLite 演示数据库（含全部演示数据，可选）
docker-compose.yml                       # 【推荐】PostgreSQL + 后端一键编排
.env.example                             # 环境变量示例（复制为 .env 后使用）
scripts/demo-reset.sh                    # 演示数据一键重置脚本（SQLite 模式）
docs/                                    # 用户手册（中英）、演示讲解手册、交付报告
DEPLOY.md                                # 本文件
```

---

## 1. 环境变量说明

完整清单见 `.env.example`。核心变量如下：

| 变量 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `APP_ENV` | 否 | `development` | `development` / `test` / `production`。**`production` 强制要求 `DB_DRIVER=postgres`**，否则启动即校验失败退出 |
| `DB_DRIVER` | 否 | `sqlite` | `sqlite` 或 `postgres`。生产必须为 `postgres` |
| `DB_DSN` | 生产必填 | SQLite 本地路径 | 数据库连接串。未设置时，若 `DB_DRIVER=postgres` 会回退读取平台注入的 `DATABASE_URL` |
| `SESSION_KEY` | **是** | `dev-session-key-change-me` | 会话签名密钥，**长度必须 ≥ 16 字符**，生产务必替换为随机值 |
| `SESSION_DAYS` | 否 | `7` | 会话有效期（天） |
| `PORT` | 否 | `8080` | 监听端口 |
| `DB_INIT` | 否 | 未设置 | **一次性初始化开关**。仅 `APP_ENV=production` 时生效：设为 `true` 才会自动建表 + 写入演示数据 |
| `GAIOS_AGENT_API_KEY` | 否 | 空 | 启用 G.AIOS 智能体联动（AI Assistant 聊天、`/api/v1/agent/*`）。留空则相关接口返回 503，**其余功能完全不受影响** |

生成安全随机值：

```bash
openssl rand -hex 32
```

> ⚠️ **安全约定**：真实密码、API Key、Secret 只写入本地 `.env`（不要提交仓库）或部署平台的
> Secret 管理。`.env.example` 中全部为占位值，可以安全提交。

---

## 方式一：Docker Compose（推荐）

一条命令同时拉起 **PostgreSQL 16** 与 **后端**，后端通过容器网络用服务名 `postgres` 连接数据库。

### 1）准备环境变量

```bash
cp .env.example .env
# 编辑 .env，至少替换这两个占位值：
#   POSTGRES_PASSWORD=...          换成强随机密码
#   SESSION_KEY=...                换成 openssl rand -hex 32 的输出
```

### 2）启动 PostgreSQL

```bash
docker compose up -d postgres
```

PostgreSQL 端口仅绑定在 `127.0.0.1`，不对外暴露。等待健康检查通过：

```bash
docker compose ps
# postgres 的 STATUS 应显示 (healthy)
```

### 3）初始化数据库并启动后端

`.env` 中 `DB_INIT=true` 时，后端**首次启动会自动建表并写入演示数据**，无需手工执行 SQL：

```bash
docker compose up -d backend
```

如需使用本地源码构建镜像（而非拉取 GHCR 镜像）：

```bash
docker compose build
docker compose up -d
```

### 4）验证

```bash
curl http://localhost:8080/health/live
```

返回 HTTP 200 即成功。浏览器打开 `http://<host>:8080/` 即可使用。

### 5）完成首次初始化后

确认系统正常后，把 `.env` 中的 `DB_INIT` 改为 `false` 并重启后端，避免每次启动都走初始化分支：

```bash
sed -i 's/^DB_INIT=true/DB_INIT=false/' .env
docker compose up -d backend
```

> 说明：`database.Seed()` 本身是幂等的（`users` 表非空即跳过），改为 `false` 只是让启动路径更干净。

### 常用运维命令

```bash
docker compose logs -f backend      # 跟踪后端日志
docker compose logs -f postgres     # 跟踪数据库日志
docker compose restart backend      # 重启后端
docker compose down                 # 停止并删除容器（保留数据卷）
docker compose down -v              # 停止并删除容器 + 数据卷（数据全部清空）
```

---

## 方式二：Docker 单容器

适用于已有托管 PostgreSQL（RDS / 云数据库），只需要跑后端。

```bash
docker run -d --name training-backend \
  -p 8080:8080 \
  -e APP_ENV=production \
  -e DB_DRIVER=postgres \
  -e DB_DSN='postgres://USER:PASSWORD@DB_HOST:5432/DBNAME?sslmode=require' \
  -e SESSION_KEY="$(openssl rand -hex 32)" \
  -e DB_INIT=true \
  ghcr.io/ziruibai/training-system:latest
```

- 需要自动建表 + 演示数据：加 `-e DB_INIT=true`（一次性）。
- 数据库已初始化好：**不要**传 `DB_INIT`（或设为 `false`）。
- 托管数据库通常要求 SSL，`sslmode=require` 按实际情况调整。

---

## 方式三：直接运行二进制

无需 Docker，适合内网离线或已有数据库的环境。

```bash
chmod +x server

# PostgreSQL 生产模式
APP_ENV=production DB_DRIVER=postgres \
  DB_DSN='postgres://USER:PASSWORD@DB_HOST:5432/DBNAME?sslmode=disable' \
  SESSION_KEY="$(openssl rand -hex 32)" \
  DB_INIT=true \
  PORT=8080 ./server
```

> ⚠️ SQLite 模式下数据文件默认写在 `var/training.db`（相对工作目录），启动前必须先 `mkdir -p var`。
> 且 `APP_ENV=production` 会拒绝 SQLite，SQLite 必须用 `APP_ENV=development`。

---

## 方式四：SQLite 快速演示

适合本地演示，**不需要 PostgreSQL**，开箱即用。

```bash
chmod +x server
mkdir -p var   # SQLite 数据文件目录，必须先创建

APP_ENV=development DB_DRIVER=sqlite PORT=8080 ./server
```

SQLite 模式下会自动建表并灌入演示数据，无需 `DB_INIT`。

重置演示数据（回到初始状态）：

```bash
bash scripts/demo-reset.sh
```

---

## 6. 数据库初始化说明

本项目提供**两条等价的初始化路径**，按你的部署方式二选一即可，**不要同时手工执行**。

### 路径 A：由应用自动初始化（Docker 与二进制部署默认）

后端在 `APP_ENV=production` 且 `DB_INIT=true` 时：

1. 调用 GORM `AutoMigrate` 依据 `model.AllModels()` 建表（共 **19 张业务表**）；
2. 调用 `database.Seed()` 写入演示数据（幂等，`users` 表非空则跳过）。

这条路径**不需要**执行任何 SQL 文件。

### 路径 B：手工执行 migration（DBA / 托管数据库场景）

若你希望人工审阅 DDL 或由 DBA 执行，按顺序执行：

```bash
psql "postgres://USER:PASSWORD@DB_HOST:5432/DBNAME?sslmode=require" \
  -f backend/migrations/postgres/001_init.sql

psql "postgres://USER:PASSWORD@DB_HOST:5432/DBNAME?sslmode=require" \
  -f backend/migrations/postgres/002_add_templates_and_coaching.sql
```

或用 Docker 一次性执行：

```bash
docker run --rm -i \
  -e PGPASSWORD="$POSTGRES_PASSWORD" \
  -v "$PWD/backend/migrations/postgres:/mig:ro" \
  postgres:16-alpine \
  psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" \
       -f /mig/001_init.sql -f /mig/002_add_templates_and_coaching.sql
```

两个文件都使用 `CREATE TABLE / INDEX IF NOT EXISTS`，**可安全重复执行**。

> **关于 002 补丁**：`001_init.sql` 只包含 16 张表，而 `model.AllModels()` 实际注册了
> 19 个模型。缺少的 3 张表（`training_templates`、`training_template_stages`、
> `coaching_recommendations`）对应「训练模板」与「辅导建议」功能，在 SQLite 下由
> `AutoMigrate` 自动补齐，但在 PostgreSQL 下若只执行 001 会导致这两个功能运行时报错。
> `002_add_templates_and_coaching.sql` 用于补齐，使两条路径的表结构完全一致。

### 演示数据

演示数据由 `backend/internal/database/seed.go` 提供（**已存在，无需另建**），包含：
9 个登录账号、5 个部门、多个岗位及能力项、11 名员工的培养计划与阶段、
学习任务/记录、阶段考核、课程、培训资料、SOP、周报月报、模板等。
两条初始化路径都会写入同一套数据。

---

## 7. 如何检查服务是否启动成功

### 健康检查接口

| 接口 | 期望 | 说明 |
|---|---|---|
| `GET /health/live` | 200 | 存活探针，进程在跑即返回 200 |
| `GET /health/ready` | 200 | 就绪探针，依赖（数据库）可用时返回 200 |

```bash
# 本机
curl -i http://localhost:8080/health/live
curl -i http://localhost:8080/health/ready

# 远程
curl -i http://<host>:8080/health/live
```

### 业务链路抽查

```bash
# 1) 演示账号列表可读取
curl -s http://localhost:8080/api/v1/auth/demo-accounts

# 2) 登录（以 Admin 为例）
curl -s -c /tmp/cj.txt -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"Admin","password":"System Admin123"}'

# 3) 带会话读取受保护资源
curl -s -b /tmp/cj.txt http://localhost:8080/api/v1/auth/me
curl -s -b /tmp/cj.txt http://localhost:8080/api/v1/employees
```

预期：前两步返回 200；未带会话访问 `/api/v1/employees` 应返回 **401**（说明鉴权生效）。

### 浏览器

打开 `http://<host>:8080/` —— 前端已内嵌在二进制中，会跳转到登录页。

### 查看容器日志

```bash
docker compose logs -f backend
# 正常启动会打印：
# training-platform listening on :8080 (env=production, db=postgres)
```

---

## 8. 演示账号

| 账号 | 密码 | 角色 |
|---|---|---|
| Admin | System Admin123 | 管理员 |
| Manager | Alice Manager123 | 经理（产品部） |
| Erin | Erin123 | 员工（AI 产品经理新人） |
| Dev | Dev123 | 员工（研发岗） |
| Sarah / Tom | Sarah123 / Tom123 | 员工（销售 / 客服岗） |
| Frank / Grace | Frank123 / Grace123 | 员工（产品部新人） |
| Bob New Hire | Bob New Hire123 | 员工（产品部新人） |

> ⚠️ 演示密码仅供试用。正式上线前请修改默认密码或关闭演示账号入口。

---

## 9. 常见问题

**Q：使用 GHCR 镜像报 `denied` / `unauthorized`？**
仓库为私有包时需先登录：

```bash
echo "$GITHUB_TOKEN" | docker login ghcr.io -u <你的GitHub用户名> --password-stdin
```

**Q：后端启动即退出，日志显示 config 校验失败？**
最常见两个原因：
1. `APP_ENV=production` 但 `DB_DRIVER` 不是 `postgres` → 按提示改用 PostgreSQL；
2. `SESSION_KEY` 少于 16 字符 → 用 `openssl rand -hex 32` 重新生成。

**Q：`DB_DSN is required` / 连不上数据库？**
- Docker Compose 内，主机名必须用服务名 `postgres`，不能用 `localhost`；
- 单容器 / 二进制部署则填数据库真实可达地址；
- 确认 `depends_on: condition: service_healthy` 已生效（compose 会自动等待）。

**Q：模板或辅导功能报错「表不存在」？**
说明数据库只执行了 `001_init.sql`。补执行 `002_add_templates_and_coaching.sql`，
或改用 `DB_INIT=true` 让 GORM 自动建表。详见[第 6 节](#6-数据库初始化说明)。

**Q：AI Assistant 聊天返回 503？**
未配置 `GAIOS_AGENT_API_KEY`。这是预期行为，配置该变量后重启即可启用。

**Q：如何完全重置演示数据？**
SQLite：`bash scripts/demo-reset.sh`；
PostgreSQL：`docker compose down -v` 后重新 `up`（会清空数据卷）。

---

## 10. 生产环境注意事项

1. **务必替换敏感值**：`SESSION_KEY`、`POSTGRES_PASSWORD` 使用随机强值，切勿沿用示例。
2. **初始化完成后设 `DB_INIT=false`**，避免每次启动都走建表 + 灌数据分支。
3. **数据库不要暴露公网**：compose 默认只绑定 `127.0.0.1`；生产建议改用托管数据库并开启 SSL。
4. **数据持久化**：compose 使用命名卷 `pgdata`；如需备份请配置定期 `pg_dump`。
5. **演示账号**：上线前修改默认密码，避免使用弱口令。
6. **会话有效期**：按安全策略调整 `SESSION_DAYS`。
7. **G.AIOS 联动**：`/api/v1/agent/*` 供 G.AIOS Script Tools 调用（`get_employee_profile` /
   `get_employee_learning_data` / `get_employee_tasks`），部署后把本服务可访问地址配置到
   G.AIOS 工具配置中，参考 `docs/GAIOS_SCRIPT_TOOL_SETUP.md`。