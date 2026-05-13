-- name: UpsertAuthorByEmail :one
-- 同じ email が既に存在する場合は name / bio / metadata を更新して返す。
-- EXCLUDED.* は INSERT しようとした新しい値を参照する。
INSERT INTO authors (name, bio, metadata, email)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email) DO UPDATE
  SET name     = EXCLUDED.name,
      bio      = EXCLUDED.bio,
      metadata = EXCLUDED.metadata
RETURNING *;

-- name: InsertAuthorIgnoreConflict :execrows
-- 同じ email が既に存在する場合は何もせず 0 行を返す。
-- 戻り値 (int64) で挿入されたか衝突したかを判別できる。
INSERT INTO authors (name, bio, metadata, email)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email) DO NOTHING;
