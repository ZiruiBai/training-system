-- Onboarding Training Platform - PostgreSQL schema (patch 002)
-- Version 2026.09.002
--
-- 背景：001_init.sql 发布于模板与辅导功能落地之前，只覆盖了 16 张表。
-- 而 GORM 的 model.AllModels() 实际注册了 19 个模型（见
-- backend/internal/model/models_competency.go），Go 后端在 SQLite 下由
-- AutoMigrate 自动补齐，但在 PostgreSQL 下若只执行 001_init.sql，
-- 以下三个功能会因为缺表而运行时报错：
--   - 训练模板（/api/v1/templates 系列接口）      -> training_templates / training_template_stages
--   - 辅导建议（/api/v1/coaching 系列接口）        -> coaching_recommendations
--
-- 本文件把上述 3 张表补齐，使「手工执行 migration」与「DB_INIT=true 由
-- GORM 建表」两条路径产出的表结构一致。
--
-- 说明：PostgreSQL 支持 CREATE TABLE / INDEX IF NOT EXISTS，因此本文件
-- 可安全重复执行；表定义与 001_init.sql 保持同一风格。
-- 注意：若你使用 DB_INIT=true 由 GORM 自动建表，则无需执行本文件。

-- ---------------------------------------------------------------------------
-- 训练模板（30/60/90 天培养计划蓝图）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS training_templates (
    id             BIGSERIAL PRIMARY KEY,
    title          VARCHAR(255) NOT NULL,
    description    TEXT,
    position_id    BIGINT REFERENCES positions(id),
    department_id  BIGINT REFERENCES departments(id),
    cycle_days     INTEGER NOT NULL DEFAULT 90,
    status         VARCHAR(16) NOT NULL DEFAULT 'draft',
    created_source VARCHAR(16) NOT NULL DEFAULT 'manual',
    ai_generated   BOOLEAN NOT NULL DEFAULT FALSE,
    ai_agent_id    VARCHAR(128),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_training_templates_deleted_at ON training_templates(deleted_at);
CREATE INDEX IF NOT EXISTS idx_training_templates_position ON training_templates(position_id);

-- ---------------------------------------------------------------------------
-- 训练模板的阶段定义（每个模板 30/60/90 各一段，含示例任务标题）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS training_template_stages (
    id                 BIGSERIAL PRIMARY KEY,
    template_id        BIGINT NOT NULL REFERENCES training_templates(id),
    stage_number       INTEGER NOT NULL,
    name               VARCHAR(128) NOT NULL,
    objectives         TEXT,
    sample_task_titles TEXT,
    start_day_offset   INTEGER NOT NULL DEFAULT 0,
    duration_days      INTEGER NOT NULL DEFAULT 30,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_training_template_stages_template_id ON training_template_stages(template_id);
CREATE INDEX IF NOT EXISTS idx_training_template_stages_deleted_at ON training_template_stages(deleted_at);

-- ---------------------------------------------------------------------------
-- 辅导建议（规则引擎生成、由经理审批后落为补充学习任务）
-- gaps / items 由 Go 侧以 JSON 序列化写入，故使用 JSONB。
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS coaching_recommendations (
    id          BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id),
    manager_id  BIGINT NOT NULL REFERENCES users(id),
    gaps        JSONB,
    plan_text   TEXT,
    items       JSONB,
    status      VARCHAR(16) NOT NULL DEFAULT 'draft',
    applied_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_coaching_recommendations_employee_id ON coaching_recommendations(employee_id);
CREATE INDEX IF NOT EXISTS idx_coaching_recommendations_manager_id ON coaching_recommendations(manager_id);