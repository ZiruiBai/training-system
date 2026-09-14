-- Onboarding Training Platform - PostgreSQL schema
-- Version 2026.08.001 (ordered, reviewable production migration)
-- Applies to PostgreSQL; development uses GORM AutoMigrate on SQLite.

CREATE TABLE IF NOT EXISTS users (
    id           BIGSERIAL PRIMARY KEY,
    username     VARCHAR(64)  NOT NULL UNIQUE,
    email        VARCHAR(128) UNIQUE,
    password     VARCHAR(255) NOT NULL,
    display_name VARCHAR(128),
    role         VARCHAR(16)  NOT NULL DEFAULT 'employee',
    employee_id  BIGINT,
    active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS departments (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL UNIQUE,
    description VARCHAR(512),
    manager_id  BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS positions (
    id                BIGSERIAL PRIMARY KEY,
    name              VARCHAR(128) NOT NULL,
    department_id     BIGINT NOT NULL REFERENCES departments(id),
    description       VARCHAR(512),
    responsibilities  TEXT,
    requirement       TEXT,
    core_abilities    TEXT,
    cycle_days        INTEGER NOT NULL DEFAULT 90,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_positions_department ON positions(department_id);

CREATE TABLE IF NOT EXISTS position_competencies (
    id          BIGSERIAL PRIMARY KEY,
    position_id BIGINT NOT NULL REFERENCES positions(id),
    name        VARCHAR(128) NOT NULL,
    target_level DOUBLE PRECISION NOT NULL DEFAULT 80,
    weight      DOUBLE PRECISION NOT NULL DEFAULT 1,
    description VARCHAR(512),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_position_competencies_position ON position_competencies(position_id);

CREATE TABLE IF NOT EXISTS employees (
    id              BIGSERIAL PRIMARY KEY,
    employee_code   VARCHAR(32) NOT NULL UNIQUE,
    name            VARCHAR(128) NOT NULL,
    email           VARCHAR(128) NOT NULL UNIQUE,
    phone           VARCHAR(32),
    department_id   BIGINT REFERENCES departments(id),
    position_id     BIGINT REFERENCES positions(id),
    manager_id      BIGINT REFERENCES users(id),
    hire_date       DATE,
    work_experience INTEGER NOT NULL DEFAULT 0,
    skill_level     VARCHAR(16) NOT NULL DEFAULT 'junior',
    current_stage   INTEGER NOT NULL DEFAULT 1,
    training_status VARCHAR(16) NOT NULL DEFAULT 'not_started',
    progress        DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_employees_department ON employees(department_id);
CREATE INDEX IF NOT EXISTS idx_employees_position ON employees(position_id);
CREATE INDEX IF NOT EXISTS idx_employees_manager ON employees(manager_id);

CREATE TABLE IF NOT EXISTS training_plans (
    id             BIGSERIAL PRIMARY KEY,
    employee_id    BIGINT NOT NULL REFERENCES employees(id),
    start_date     DATE,
    end_date       DATE,
    cycle_days     INTEGER NOT NULL DEFAULT 90,
    current_stage  INTEGER NOT NULL DEFAULT 1,
    progress       DOUBLE PRECISION NOT NULL DEFAULT 0,
    status         VARCHAR(16) NOT NULL DEFAULT 'draft',
    reviewed_by    BIGINT REFERENCES users(id),
    reviewed_at    TIMESTAMPTZ,
    created_by     BIGINT REFERENCES users(id),
    created_source VARCHAR(16) NOT NULL DEFAULT 'manual',
    ai_generated   BOOLEAN NOT NULL DEFAULT FALSE,
    ai_agent_id    VARCHAR(128),
    ai_session_id  VARCHAR(128),
    ai_workflow_id VARCHAR(128),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_training_plans_employee ON training_plans(employee_id);
CREATE INDEX IF NOT EXISTS idx_training_plans_status ON training_plans(status);

CREATE TABLE IF NOT EXISTS training_stages (
    id           BIGSERIAL PRIMARY KEY,
    plan_id      BIGINT NOT NULL REFERENCES training_plans(id),
    stage_number INTEGER NOT NULL,
    name         VARCHAR(128) NOT NULL,
    objectives   TEXT,
    start_date   DATE,
    end_date     DATE,
    status       VARCHAR(16) NOT NULL DEFAULT 'not_started',
    progress     DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_training_stages_plan ON training_stages(plan_id);

CREATE TABLE IF NOT EXISTS learning_tasks (
    id             BIGSERIAL PRIMARY KEY,
    employee_id    BIGINT NOT NULL REFERENCES employees(id),
    plan_id        BIGINT NOT NULL REFERENCES training_plans(id),
    stage_id       BIGINT REFERENCES training_stages(id),
    title          VARCHAR(255) NOT NULL,
    description    TEXT,
    task_type      VARCHAR(16) NOT NULL,
    course_id      BIGINT,
    material_id    BIGINT,
    start_date     DATE,
    due_date       DATE,
    estimate_hours DOUBLE PRECISION,
    priority       VARCHAR(8) NOT NULL DEFAULT 'medium',
    status         VARCHAR(16) NOT NULL DEFAULT 'pending',
    completed_at   TIMESTAMPTZ,
    score          DOUBLE PRECISION,
    outcome        TEXT,
    remark         VARCHAR(512),
    created_by     BIGINT,
    created_source VARCHAR(16) NOT NULL DEFAULT 'manual',
    ai_generated   BOOLEAN NOT NULL DEFAULT FALSE,
    ai_agent_id    VARCHAR(128),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_learning_tasks_employee ON learning_tasks(employee_id);
CREATE INDEX IF NOT EXISTS idx_learning_tasks_plan ON learning_tasks(plan_id);
CREATE INDEX IF NOT EXISTS idx_learning_tasks_stage ON learning_tasks(stage_id);
CREATE INDEX IF NOT EXISTS idx_learning_tasks_status ON learning_tasks(status);

CREATE TABLE IF NOT EXISTS learning_records (
    id           BIGSERIAL PRIMARY KEY,
    employee_id  BIGINT NOT NULL REFERENCES employees(id),
    task_id      BIGINT NOT NULL REFERENCES learning_tasks(id),
    content      TEXT,
    start_time   TIMESTAMPTZ,
    end_time     TIMESTAMPTZ,
    duration_min INTEGER NOT NULL DEFAULT 0,
    status       VARCHAR(16) NOT NULL DEFAULT 'pending',
    test_score   DOUBLE PRECISION,
    self_eval    VARCHAR(1024),
    manager_eval VARCHAR(1024),
    outcome      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_learning_records_employee ON learning_records(employee_id);
CREATE INDEX IF NOT EXISTS idx_learning_records_task ON learning_records(task_id);

CREATE TABLE IF NOT EXISTS assessments (
    id               BIGSERIAL PRIMARY KEY,
    employee_id      BIGINT NOT NULL REFERENCES employees(id),
    plan_id          BIGINT NOT NULL REFERENCES training_plans(id),
    stage_number     INTEGER NOT NULL,
    assess_date      DATE,
    assessor_id      BIGINT REFERENCES users(id),
    composite_score  DOUBLE PRECISION,
    strengths        TEXT,
    weaknesses       TEXT,
    improvement      TEXT,
    goals_met        BOOLEAN NOT NULL DEFAULT FALSE,
    status           VARCHAR(16) NOT NULL DEFAULT 'draft',
    assessment_source VARCHAR(16) NOT NULL DEFAULT 'manual',
    ai_generated     BOOLEAN NOT NULL DEFAULT FALSE,
    ai_agent_id      VARCHAR(128),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_assessments_employee ON assessments(employee_id);

CREATE TABLE IF NOT EXISTS assessment_scores (
    id            BIGSERIAL PRIMARY KEY,
    assessment_id BIGINT NOT NULL REFERENCES assessments(id),
    capability    VARCHAR(128) NOT NULL,
    score         DOUBLE PRECISION NOT NULL,
    target_level  DOUBLE PRECISION NOT NULL DEFAULT 80,
    weight        DOUBLE PRECISION NOT NULL DEFAULT 1,
    remark        VARCHAR(512),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_assessment_scores_assessment ON assessment_scores(assessment_id);

CREATE TABLE IF NOT EXISTS courses (
    id             BIGSERIAL PRIMARY KEY,
    title          VARCHAR(255) NOT NULL,
    category       VARCHAR(16) NOT NULL,
    position_id    BIGINT REFERENCES positions(id),
    description    TEXT,
    objectives     TEXT,
    content        TEXT,
    difficulty     INTEGER NOT NULL DEFAULT 1,
    estimate_hours DOUBLE PRECISION,
    status         VARCHAR(16) NOT NULL DEFAULT 'published',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS training_materials (
    id          BIGSERIAL PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    file_type   VARCHAR(16),
    position_id BIGINT REFERENCES positions(id),
    course_id   BIGINT REFERENCES courses(id),
    category    VARCHAR(16) NOT NULL DEFAULT 'general',
    description TEXT,
    file_path   VARCHAR(512),
    uploader_id BIGINT,
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sop_documents (
    id            BIGSERIAL PRIMARY KEY,
    title         VARCHAR(255) NOT NULL,
    sop_type      VARCHAR(16) NOT NULL,
    department_id BIGINT REFERENCES departments(id),
    position_id   BIGINT REFERENCES positions(id),
    version       VARCHAR(32) NOT NULL DEFAULT '1.0',
    content       TEXT,
    file_path     VARCHAR(512),
    effective_at  DATE,
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS training_reports (
    id             BIGSERIAL PRIMARY KEY,
    employee_id    BIGINT NOT NULL REFERENCES employees(id),
    plan_id        BIGINT REFERENCES training_plans(id),
    report_type    VARCHAR(16) NOT NULL,
    period         VARCHAR(64),
    stage_number   INTEGER,
    progress       DOUBLE PRECISION,
    task_completion DOUBLE PRECISION,
    learning_hours DOUBLE PRECISION,
    composite_score DOUBLE PRECISION,
    capabilities   TEXT,
    strengths      TEXT,
    weaknesses     TEXT,
    risks          TEXT,
    suggestions    TEXT,
    manager_eval   TEXT,
    content        TEXT,
    ai_generated   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_training_reports_employee ON training_reports(employee_id);

CREATE TABLE IF NOT EXISTS sessions (
    id         BIGSERIAL PRIMARY KEY,
    token      VARCHAR(64) NOT NULL UNIQUE,
    user_id    BIGINT NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);