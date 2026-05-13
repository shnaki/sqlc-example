-- name: CopyInsertPosts :copyfrom
-- :copyfrom は pgx の CopyFrom プロトコルを使った高速バルク INSERT。
-- (:batchexec との違い) batchexec は複数 INSERT を1往復でまとめるが
-- copyfrom は PostgreSQL COPY プロトコルを使うためさらに高速。
-- 戻り値は挿入件数 (int64)。
INSERT INTO posts (author_id, title, body, tags, status)
VALUES ($1, $2, $3, $4, $5);
