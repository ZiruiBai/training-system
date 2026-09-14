# G.AIOS Script Tool 配置清单（可直接粘贴）

> 生成时间：2026-08-27
> 用途：在 G.AIOS 后台创建 3 个 Script Tool，使 Agent 能读取 WorkOS 中**当前登录员工**的真实业务数据。
> 身份来源：`runtime.userInfo.employeeId / role / userId`（由前端 embed.js 注入，**不要写死、不要解析 user_id**）。

---

## 0. 允许主机（在工具配置/全局配置里声明）

```
<你的后端域名>
```
> 用**固定生产域名**最佳；若用临时预览域名，过期后需同步更新。当前测试预览为 `https://5oeyn.dev.gazellio.com`（会过期，仅用于测试）。

---

## 1. get_employee_profile

- **工具名**：`get_employee_profile`
- **描述**：根据当前登录员工获取个人档案（姓名/部门/岗位/入职时间/当前阶段/培养状态）。
- **输入 JSON Schema**：
```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```
- **脚本全文**：
```javascript
function main(input, runtime) {
  const base = '<你的后端域名>';
  const url = base + '/api/v1/agent/get_employee_profile';
  const response = fetch(url, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      'X-User-Role': String(runtime.userInfo.role || ''),
      'X-Employee-Id': String(runtime.userInfo.employeeId || ''),
      'X-User-Id': String(runtime.userInfo.userId || '')
    }
  });
  return { status: response.status, data: response.json() };
}
```
- **返回字段**（后端已定义）：`employee_id, name, department, position, join_date, current_stage, status`

---

## 2. get_employee_learning_data

- **工具名**：`get_employee_learning_data`
- **描述**：获取当前登录员工的培养计划进度、任务完成情况、课程与成绩、各项能力评分、历史评估，用于学习情况/能力短板分析。
- **输入 JSON Schema**：
```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```
- **脚本全文**：
```javascript
function main(input, runtime) {
  const base = '<你的后端域名>';
  const url = base + '/api/v1/agent/get_employee_learning_data';
  const response = fetch(url, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      'X-User-Role': String(runtime.userInfo.role || ''),
      'X-Employee-Id': String(runtime.userInfo.employeeId || ''),
      'X-User-Id': String(runtime.userInfo.userId || '')
    }
  });
  return { status: response.status, data: response.json() };
}
```
- **返回字段**（后端已定义）：
  `employee_id, plan_id, plan_status, current_stage, plan_progress, total_tasks, completed_tasks, incomplete_tasks, overdue_tasks, total_courses, completed_courses, course_rate, test_scores[], avg_test_score, capabilities[{capability,score,target_level,met}], assessments[{stage_number,assess_date,composite_score,goals_met,status}]`

---

## 3. get_employee_tasks

- **工具名**：`get_employee_tasks`
- **描述**：获取当前登录员工的最近学习任务清单（标题/类型/截止日/状态/得分）。
- **输入 JSON Schema**：
```json
{
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "description": "返回任务条数，默认10，最大50",
      "default": 10,
      "minimum": 1,
      "maximum": 50
    }
  },
  "additionalProperties": false
}
```
- **脚本全文**：
```javascript
function main(input, runtime) {
  const base = '<你的后端域名>';
  const limit = input.limit || 10;
  const url = base + '/api/v1/agent/get_employee_tasks?limit=' + encodeURIComponent(limit);
  const response = fetch(url, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      'X-User-Role': String(runtime.userInfo.role || ''),
      'X-Employee-Id': String(runtime.userInfo.employeeId || ''),
      'X-User-Id': String(runtime.userInfo.userId || '')
    }
  });
  return { status: response.status, data: response.json() };
}
```
- **返回字段**：`data[{ task_id, title, description, task_type, due_date, status, score }]`

---

## 4. Agent Prompt 建议（可选，帮助 Agent 会用这些工具）

> 你可以在智能体的系统提示里加这段：
> - 当需要员工个人/学习/任务信息时，调用 `get_employee_profile`、`get_employee_learning_data`、`get_employee_tasks`。
> - 已通过 userInfo 注入当前员工身份，**无需向用户索要员工ID**。
> - 用户问「今天有哪些学习任务」→ 调 `get_employee_tasks` 后结合 due_date 筛选今天。

---

## 5. 前端 userInfo 字段映射（GAIOSAgentEmbed 已注入）

| userInfo 字段 | 来源 | 说明 |
|---|---|---|
| `userId` | `session.user_id` | 业务用户ID，manager 行级鉴权用 |
| `userName` | `session.display_name` | 显示名 |
| `employeeId` | `session.employee_id` | **员工数字ID，行级鉴权关键** |
| `role` | `session.role` | `employee/manager/admin` |