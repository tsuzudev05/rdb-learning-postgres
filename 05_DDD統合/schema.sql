-- =============================================================================
-- OKR Management Tool - Database Schema
-- =============================================================================
-- DDD / Clean Architecture 対応
-- PostgreSQL 16
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Extensions
-- -----------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -----------------------------------------------------------------------------
-- users
-- 集約ルート: User
-- -----------------------------------------------------------------------------
CREATE TABLE users (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  users              IS 'ユーザー（集約ルート: User）';
COMMENT ON COLUMN users.id           IS 'ユーザーID（UserId値オブジェクト）';
COMMENT ON COLUMN users.email        IS 'メールアドレス（一意制約）';
COMMENT ON COLUMN users.password_hash IS 'ハッシュ化済みパスワード';

-- -----------------------------------------------------------------------------
-- teams
-- 集約ルート: Team
-- -----------------------------------------------------------------------------
CREATE TABLE teams (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  teams             IS 'チーム（集約ルート: Team）';
COMMENT ON COLUMN teams.id          IS 'チームID（TeamId値オブジェクト）';

-- -----------------------------------------------------------------------------
-- team_members
-- エンティティ: TeamMember（Team集約内）
-- team × user の中間テーブル。role によりドメインルールを区別する。
-- -----------------------------------------------------------------------------
CREATE TABLE team_members (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id     UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        VARCHAR(20) NOT NULL DEFAULT 'member'
                    CHECK (role IN ('admin', 'member')),
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (team_id, user_id)
);

COMMENT ON TABLE  team_members          IS 'チームメンバー（Team集約内エンティティ）';
COMMENT ON COLUMN team_members.role     IS 'ロール: admin | member';
COMMENT ON COLUMN team_members.team_id  IS 'Team集約ルートへの参照';

-- -----------------------------------------------------------------------------
-- periods
-- 集約ルート: Period（半期単位の OKR サイクル）
-- -----------------------------------------------------------------------------
CREATE TABLE periods (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id     UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name        VARCHAR(50) NOT NULL,           -- 例: "2025-H1"
    half        VARCHAR(2)  NOT NULL
                    CHECK (half IN ('H1', 'H2')),
    start_date  DATE        NOT NULL,
    end_date    DATE        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (start_date < end_date)
);

COMMENT ON TABLE  periods            IS '半期（集約ルート: Period）';
COMMENT ON COLUMN periods.half       IS '上期 H1 / 下期 H2';
COMMENT ON COLUMN periods.team_id    IS 'どのチームの期間か';

-- -----------------------------------------------------------------------------
-- objectives
-- 集約ルート: Objective（OKR の O）
-- owner_id がそのチームのメンバーかはドメインサービス層で検証する。
-- -----------------------------------------------------------------------------
CREATE TABLE objectives (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    period_id     UUID        NOT NULL REFERENCES periods(id) ON DELETE CASCADE,
    owner_id      UUID        NOT NULL REFERENCES users(id),
    title         VARCHAR(255) NOT NULL,
    description   TEXT,
    display_order INT         NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  objectives               IS '目標（集約ルート: Objective）';
COMMENT ON COLUMN objectives.owner_id      IS 'オーナーUser。チームメンバー整合性はドメインサービスで保証';
COMMENT ON COLUMN objectives.display_order IS 'UI上の表示順（ドラッグ並び替え対応）';

-- -----------------------------------------------------------------------------
-- key_results
-- 集約ルート: KeyResult（OKR の KR）
-- kr_type により NumericProgress / CheckboxProgress に分岐。
-- target_value / current_value は kr_type='checkbox' の場合 NULL を許容。
-- -----------------------------------------------------------------------------
CREATE TABLE key_results (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    objective_id    UUID        NOT NULL REFERENCES objectives(id) ON DELETE CASCADE,
    owner_id        UUID        NOT NULL REFERENCES users(id),
    title           VARCHAR(255) NOT NULL,
    kr_type         VARCHAR(20) NOT NULL
                        CHECK (kr_type IN ('numeric', 'checkbox')),
    -- numeric 型のみ使用
    target_value    NUMERIC,
    current_value   NUMERIC,
    -- checkbox 型のみ使用
    is_completed    BOOLEAN     NOT NULL DEFAULT FALSE,
    display_order   INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- numeric の場合 target_value は必須
    CHECK (
        (kr_type = 'numeric' AND target_value IS NOT NULL)
        OR kr_type = 'checkbox'
    )
);

COMMENT ON TABLE  key_results                  IS '主要な結果（集約ルート: KeyResult）';
COMMENT ON COLUMN key_results.kr_type          IS '進捗種別: numeric（数値） | checkbox（完了フラグ）';
COMMENT ON COLUMN key_results.target_value     IS '目標値。kr_type=numeric のみ使用';
COMMENT ON COLUMN key_results.current_value    IS '現在値。kr_type=numeric のみ使用';
COMMENT ON COLUMN key_results.is_completed     IS '完了フラグ。kr_type=checkbox のみ使用';
COMMENT ON COLUMN key_results.display_order    IS 'UI上の表示順';

-- -----------------------------------------------------------------------------
-- kr_progress_logs
-- エンティティ: KrProgressLog（KeyResult集約内）
-- 進捗の時系列履歴。KeyResult 本体は最新値のみ保持する。
-- -----------------------------------------------------------------------------
CREATE TABLE kr_progress_logs (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_result_id   UUID        NOT NULL REFERENCES key_results(id) ON DELETE CASCADE,
    recorded_by     UUID        NOT NULL REFERENCES users(id),
    value           NUMERIC,                -- numeric 型 KR の記録値
    completed       BOOLEAN,               -- checkbox 型 KR の記録値
    note            TEXT,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  kr_progress_logs               IS '進捗履歴（KeyResult集約内エンティティ）';
COMMENT ON COLUMN kr_progress_logs.value         IS '数値進捗。kr_type=numeric のみ';
COMMENT ON COLUMN kr_progress_logs.completed     IS '完了フラグ。kr_type=checkbox のみ';
COMMENT ON COLUMN kr_progress_logs.recorded_by   IS '記録者（UserId）';

-- -----------------------------------------------------------------------------
-- Indexes
-- -----------------------------------------------------------------------------
CREATE INDEX idx_team_members_team_id       ON team_members(team_id);
CREATE INDEX idx_team_members_user_id       ON team_members(user_id);
CREATE INDEX idx_periods_team_id            ON periods(team_id);
CREATE INDEX idx_objectives_period_id       ON objectives(period_id);
CREATE INDEX idx_objectives_owner_id        ON objectives(owner_id);
CREATE INDEX idx_key_results_objective_id   ON key_results(objective_id);
CREATE INDEX idx_kr_progress_logs_kr_id     ON kr_progress_logs(key_result_id);
CREATE INDEX idx_kr_progress_logs_recorded  ON kr_progress_logs(recorded_at);

-- -----------------------------------------------------------------------------
-- updated_at 自動更新トリガー
-- -----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_periods_updated_at
    BEFORE UPDATE ON periods
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_objectives_updated_at
    BEFORE UPDATE ON objectives
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_key_results_updated_at
    BEFORE UPDATE ON key_results
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
