-- ─────────────────────────────────────────────────────────
-- 学習用初期スキーマ
-- rdb-learning-postgres / 01_基礎
-- ─────────────────────────────────────────────────────────

-- ユーザーテーブル
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100)  NOT NULL,
    email      VARCHAR(255)  UNIQUE NOT NULL,
    created_at TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- タスクテーブル（usersと外部キー）
CREATE TABLE IF NOT EXISTS tasks (
    id         SERIAL PRIMARY KEY,
    user_id    INT           REFERENCES users(id) ON DELETE CASCADE,
    title      VARCHAR(200)  NOT NULL,
    status     VARCHAR(20)   DEFAULT 'todo'
                             CHECK (status IN ('todo', 'in_progress', 'done')),
    created_at TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);

-- サンプルデータ投入（冪等：既存データがあればスキップ）
INSERT INTO users (name, email) VALUES
    ('田中太郎', 'tanaka@example.com'),
    ('鈴木花子', 'suzuki@example.com'),
    ('佐藤次郎', 'sato@example.com')
ON CONFLICT (email) DO NOTHING;

INSERT INTO tasks (user_id, title, status)
SELECT u.id, t.title, t.status
FROM (VALUES
    ('tanaka@example.com', 'PostgreSQL環境構築',  'done'),
    ('tanaka@example.com', 'SELECT文を学ぶ',       'todo'),
    ('suzuki@example.com', 'JOINを理解する',       'todo'),
    ('sato@example.com',   'インデックスを学ぶ',   'todo')
) AS t(email, title, status)
JOIN users u ON u.email = t.email
WHERE NOT EXISTS (
    SELECT 1 FROM tasks WHERE title = t.title
);
