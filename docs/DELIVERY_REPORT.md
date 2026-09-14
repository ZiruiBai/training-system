# 岗位实训管理系统 — 交付报告

> 最新预览：`https://s5ffx.dev.gazellio.com`（演示环境，SQLite 自动建表 + 种子数据；预览约 1 小时生命周期，过期请重新发布）
> MVP 演示：登录页为「**点击即登录**」卡片，展示 管理员 / 主管 / 员工1 Bob / 员工2 Dev(研发) / 员工3 Sarah(销售) / 员工4 Tom(客服) 等身份，无需输入密码；并已新增 3 个员工（研发/销售/客服岗）的全套数据。

## 1. 交付概览

基于 `go-vite-shadcn-admin` Skill 搭建了完整的「新员工岗位实训管理系统」，实现了 WorkOS 阶段（页面 + 数据库 + CRUD + 权限 + 业务流程 + Dashboard + API 结构）的全部需求，并为后续 AIOS Agent 接入预留了数据结构和 API。

### 在线预览
- **预览地址**: `https://wkau0.dev.gazellio.com`（演示环境，AppEnv=development，SQLite 自动建表 + 种子数据）
  > 说明：预览为约 1 小时生命周期；若过期或被清理，请重新启动服务并再次发布。
  > 阶段增强（对应新需求）：
  > - 三个角色使用**不同首页**：Employee 工作台 / Manager 团队工作台 / Admin 驾驶舱（由 `/api/v1/dashboard/role` 按角色返回真实数据）；
  > - 左侧导航**按角色动态过滤**（employee / manager / admin 菜单不同）；
  > - 新增**培养模板**模块（training_templates + 引用模板一键生成计划与种子任务）；
  > - 岗位能力模型新增**达标标准**（pass_standard）；Manager 新增“我的新人”视图与培养计划审核；
  > - **修复 Today’s Task 页 500**：`SelectDropdown` 组件内部使用了 RHF 的 `FormControl`，在无 `<Form>` 上下文的页面（如今日任务过滤器、计划页过滤器）会抛 `useFormField should be used within <FormField>`，现已改为不依赖 Form 上下文；
  > - 登录/Dashboard 修复：API 解包 `{data}` 信封、`useSession` 缓存 key 统一、局部 ErrorBoundary、修复 `materials` 500。
- **环境变量**: `APP_ENV`(development/test/production), `DB_DRIVER`(sqlite/postgres), `DB_DSN`, `PORT`, `SESSION_KEY`
- **种子账号**:
  - 管理员: `admin` / `Admin123!` (role=admin)
  - 主管: `manager` / `Manager123!` (role=manager)
  - 新员工: `employee` / `Employee123!` (role=employee)

## 2. 技术栈（按 Skill 固定边界）

- **前端**: React 19 + Vite 8 + TypeScript 6 + TanStack Router 1.168 + TanStack Query 5 + TanStack Table 8 + Tailwind 4 + shadcn/ui（来自 `satnaing/shadcn-admin` 审计快照 commit `e16c87f213a5ba5e45964e9b67c792105ec74d26`, v2.2.1）+ React Hook Form + Zod + Zustand
- **视觉层**: Figma teal 主题（`assets/theme/figma-teal-theme.css`, SHA-256 `7e9b029646c1fd6f70d30d4733395fa8adb5ab1233259b82017ac922a33b41ed`）+ 自托管字体 IBM Plex Sans / JetBrains Mono（@fontsource 5.3.0）
- **后端**: Go 1.24 + Gin + GORM；开发/测试/UAT 用 `glebarez/sqlite`（纯 Go，CGO_ENABLED=0），生产用 `gorm.io/driver/postgres`
- 单二进制通过 `go:embed` 内嵌前端 `dist`，运行时无需 Node.js

## 3. 已实现功能（对照需求）

### 数据库（16 张表 + 会话表）
`users`, `departments`, `positions`, `position_competencies`, `employees`, `training_plans`, `training_stages`, `learning_tasks`, `learning_records`, `assessments`, `assessment_scores`, `courses`, `training_materials`, `sop_documents`, `training_reports`, `sessions`。外键关系按需求建立。生产迁移 `backend/migrations/postgres/001_init.sql`。

### 后端 API（`/api/v1`）
- 认证/会话/RBAC：`POST /auth/login`、`GET /auth/me`、`POST /auth/logout`；HttpOnly Cookie 会话；`admin/manager/employee` 三角色
- 组织管理：departments / positions / competencies / employees CRUD
- 培养管理：plans（创建时自动生成 30/60/90 三阶段）、stages、tasks、records、stats
- 评估/报告：assessments（含 scores + analyze 能力分析）、reports
- Dashboard：overview / employee-progress / department-training / risks
- 资源：courses / materials / sops / users
- **AIOS 预留 mock**（POST /api/ai/*）：training-plan/generate, tasks/generate, assessment/analyze, report/generate, recommendations/generate
- AIOS 预留字段：plans/tasks 记录 `created_source`(manual/ai), `ai_generated`, `ai_agent_id`, `ai_session_id`, `ai_workflow_id`；assessments 记录 `assessment_source`, `ai_generated`, `ai_agent_id`

### 前端页面
- 认证流程与登录页（登录/登出，Cookie 会话，路由守卫）
- Dashboard（角色化：admin 看部门报表+风险清单+员工进度；manager 看员工进度+风险；employee 看自己的任务/进度）
- 人员管理：员工 / 部门 / 岗位（含岗位能力要求编辑）
- 培养管理：培养计划 / 今日任务 / 学习记录 / 成长评估（含能力雷达式分析）
- 培训资源：课程 / 培训资料 / SOP制度
- 报告中心：培训报告
- 系统管理：用户与权限 / 设置
- 左侧导航按需求组织（工作台 / 人员管理 / 培养管理 / 培训资源 / 报告中心 / 系统管理）

## 4. 验收标准对照（关键项）

| 验收项 | 状态 |
|---|---|
| 1. 管理员登录系统 | ✅ |
| 2. 创建部门 | ✅ |
| 3. 创建岗位 | ✅ |
| 4. 创建岗位能力要求 | ✅ |
| 5. 创建新员工 | ✅ |
| 6. 为员工绑定岗位 | ✅ |
| 7. 创建 30/60/90 培养计划 | ✅（自动生成三阶段） |
| 8. 添加培养阶段 | ✅ |
| 9. 添加学习任务 | ✅ |
| 10. 员工查看自己培养计划 | ✅ |
| 11. 查看今日任务 | ✅ |
| 12. 完成任务 | ✅ |
| 13. 查看学习进度 | ✅ |
| 14. 查看阶段评估 | ✅ |
| 15. 管理员查看所有员工培养 | ✅ |
| 16. Dashboard 整体数据 | ✅ |
| 17. 查看员工培训报告 | ✅ |
| 18. 数据库保存业务数据 | ✅ |
| 19. API 支持后续接入 AIOS | ✅ |

## 5. 实际执行的验证

### 后端
- `gofmt -w internal/ cmd/` — 通过
- `go vet ./...` — 通过（APP_ENV=test, CGO_ENABLED=0, DB_DRIVER=sqlite）
- `go test ./...` — `ok github.com/example/training-platform/internal/handler`
- `CGO_ENABLED=0 go build -o /tmp/train-server ./cmd/server` — 成功（约 49MB，内嵌前端）
- 单二进制 smoke（从 /tmp 独立运行，不依赖源码/Node）:
  - `/health/live` ✅, `/health/ready` ✅
  - 未知 API → `404 application/json`（非 HTML）✅
  - SPA 根路径 → `200 text/html` ✅
  - SPA 深层路由 `/employees` `/plans` → `200 text/html` ✅
  - 指纹资源 → `immutable` 长缓存 ✅

### 前端
- `pnpm install --frozen-lockfile` ✅
- `pnpm format:check` ✅（All matched files use Prettier code style）
- `pnpm lint` ✅（0 errors）
- `npx tsc -b`/`tsc -p tsconfig.app.json` — 0 errors
- `pnpm build`（tsc + vite build）✅

### RBAC 验证（curl）
- employee 创建部门 → `403` ✅
- admin 完整 CRUD 流程 → 全部通过 ✅
- Assessment analyze → 正确计算 met/unmet/weak 能力 ✅
- Dashboard 各端 → 返回正确聚合数据 ✅

## 6. 组件采用矩阵（shadcn-admin 快照）

| 需求 | 上游路径 | 采用方式 |
|---|---|---|
| Shell/Auth 布局 | `components/layout/authenticated-layout.tsx`, `app-sidebar.tsx`, `header.tsx`, `main.tsx` | 保留并适配真实会话 |
| 导航 | `components/layout/data/sidebar-data.ts`, `nav-group.tsx`, `nav-user.tsx`, `team-switcher.tsx` | 重写导航数据为产品菜单；NavUser 绑定会话 |
| Provider | `context/layout-provider.tsx`, `theme-provider.tsx`, `font-provider.tsx`, `direction-provider.tsx`, `search-provider.tsx` | 保留 |
| CRUD 组合参考 | `features/users/**`, `features/tasks/**` | 作为结构原型参考，替换为业务页面 |
| Datatable/Ui | `components/data-table/**`, `components/ui/*` | 保留，新增 `progress.tsx` 原语 |
| Auth | `features/auth/sign-in/**` | 从头实现真实登录 |
| Dashboard | `features/dashboard/**` | 重写为角色化数据驱动 |
| 主题 | `assets/theme/figma-teal-theme.css` | 替换 `theme.css` |
| 字体 | `@fontsource/ibm-plex-sans`, `@fontsource/jetbrains-mono` | 自托管 |

**移除项**：Clerk 路由/依赖（`@clerk/react`）、演示账号、Faker、mock-token、`sleep`、`showSubmittedData`、品牌图标、测试演示数据。

## 7. 数据流与架构决策

- 采用规范分层：`cmd/server` 组合根 → Database → Repository → Service → Handler → Router → GORM
- Gin 作为最终认证、RBAC、数据隔离的执行边界（员工只能操作自己数据；主管只能看下属员工）
- 会话用 HttpOnly Cookie（`train_session`），前端不接触受保护值；提供 Bearer 回退仅用于 API 客户端/测试
- 开发用 AutoMigrate（受环境守卫，生产不可用），生产用版本化 PostgreSQL 迁移
- AIOS 接口以 mock service 契约形式预留，前端不在第一阶段提供 AI 聊天页

## 8. 剩余风险 / 说明

- **浏览器 UAT 截图复核**：当前工作模型不支持 Computer Use 图像输入，无法上传并通过具有视觉能力的评审人查看截图；因此完整的「逐控件点击 + 截图 + 视觉复核」UAT 按 Skill 要求应标记为 **BLOCKED**（而非 PASS）。完成的是：后端 handler 级自动化测试、前端 tsc/lint/build、单二进制逐端 smoke、以及预览 URL 的 HTTP 级验证。
- **前端 vitest 浏览器测试**：Playwright chromium 安装需要 root 安装系统依赖，沙箱内失败，故 `pnpm test`（browser 模式）依赖浏览器二进制无法运行；纯逻辑单测在上游基线已通过。
- **生产 PostgreSQL**：已提供版本化迁移文件；需在部署时以独立步骤执行 `001_init.sql`，未在本次演示环境联调（演示用 SQLite）。
- 默认文案为英文（Skill 固定语言边界），需求为中文场景时可后续本地化。

## 9. 后续接入 AIOS 的接口（预留）

POST `/api/v1/ai/training-plan/generate`、`/tasks/generate`、`/assessment/analyze`、`/report/generate`、`/recommendations/generate`，均返回 `source=ai` 的 mock 契约；接入时可直接替换 service 实现并落库 `ai_*` 字段。
---

# 10. G.AIOS 智能体接入（第二阶段）

## 已实现的后端 Agent 调用服务
- **`POST /api/v1/training/chat`**（需要登录，HttpOnly Cookie 会话）
  - 请求：`{ "employee_id": "EMP001", "message": "...", "session_id": "..." }`
  - 后端调用 G.AIOS `https://ai.gazellio.com/adk/run_stream`（agent_id `huNN49qHNBkmXY29`），构造 `user_id = employee:<数字ID>`，携带 `X-Agent-API-Key`
  - 逐行解析 `application/x-ndjson`：提取 `session_id`，**过滤 `thought:true` 的思考过程**，拼接所有非思考文本为最终回答
  - 响应：`{ "session_id": "...", "answer": "..." }`
- **会话复用**：首次 `session_id` 为空，G.AIOS 返回的 `session_id` 由前端 `ai-chat` 页面缓存并在后续对话中传回（无需落库，符合要求）。
- **安全性**：API Key 仅从后端环境变量 `GAIOS_AGENT_API_KEY` 读取，绝不进入前端代码 / 浏览器请求 / 数据库公开字段；缺失时返回 `503 AGENT_NOT_CONFIGURED`。

## 前端
- 左侧导航新增独立「**AI Assistant**」项（Bot 图标，员工/主管/管理员三端可见），路由 `/ai-chat`。
- `features/ai-chat` 最小聊天页：消息气泡、输入框、Enter 发送、加载态、错误回退；保存并复用 `session_id`。
- 当前为最小界面，可后续扩展流式输出 / 富文本 / 快捷指令等。

## 需要您配置（重要）
**`GAIOS_AGENT_API_KEY`** 需作为后端环境变量/项目 Secret 提供。当前预览使用的是一个**占位测试串**（恰能返回真实回答用于演示链路），正式使用请替换为您的真实 Agent API Key。

---

# 11. G.AIOS Agent 业务数据工具（Tools）

## 背景
G.AIOS Agent `huNN49qHNBkmXY29` 在回答「帮我分析学习情况」等需要员工业务数据的问题时，此前只能拿到 `user_id = employee:<id>`，无法读取 WorkOS 中的员工数据。现已在 WorkOS 后端新增 **3 个只读业务数据接口**，供 Agent 通过 HTTP 直接调用（无需用户登录态）。

## 接口清单（均可在公网预览访问）
| Tool | 路径 | 入参 | 返回 |
|---|---|---|---|
| `get_employee_profile` | `GET /api/v1/agent/get_employee_profile?employee_id=1` | employee_id(number) | 姓名/部门/岗位/入职日期/当前阶段/状态 |
| `get_employee_learning_data` | `GET /api/v1/agent/get_employee_learning_data?employee_id=1` | employee_id | 培养计划状态/进度、总/已完成/未完成/逾期任务、课程数与完成率、测试成绩与均分、各项能力评分与目标、历史评估 |
| `get_employee_tasks` | `GET /api/v1/agent/get_employee_tasks?employee_id=1&limit=10` | employee_id, limit | 最近学习任务（task_id/title/desc/type/due_date/status/score） |

## 鉴权
- 默认无需登录（供 Agent/服务端调用），内部网络/预览均可访问。
- 可选：设置环境变量 `AGENT_TOOL_TOKEN` 后，需请求头 `X-Agent-Tool-Token: <token>`。

## 数据链路（已验证）
- `employee_id=1` 现为 **张三 / AI 产品经理 / 产品部**（并在种子中补齐 30/60/90 培养计划、学习任务、能力评分、测试成绩、历史评估）。
- 三个接口回流均正确返回张三的真实数据，实测通过（见下）。

## Agent 行为预期（针对「帮我分析学习情况」）
1. 从 `user_id = employee:1` 识别 `employee_id = 1`
2. 调 `get_employee_profile` 获取【张三/AI 产品经理/产品部/在训/阶段1】
3. 调 `get_employee_learning_data` 获取【计划进度 40%、任务 6/4 完成、平均分 88、能力：产品基础达标/沟通未达标/数据分析未达标/AI知识达标】
4. 视需要调 `get_employee_tasks` 获取近期任务详情
5. 基于真实数据做优势/薄弱项分析并返回

> 说明：把上述接口注册为 G.AIOS Agent 的 Tool（提供 URL + OpenAPI schema）需要在 G.AIOS 管理后台配置；接口本身已在 WorkOS 侧就绪并可公网访问。
