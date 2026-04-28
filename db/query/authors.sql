-- 01-basics で使用するクエリ
-- 示す機能: :one / :many / :exec / :execrows

-- name: CreateAuthor :one
-- INSERT ... RETURNING * → Author 構造体を返す (:one)
INSERT INTO authors (name, bio, metadata)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAuthor :one
-- 主キーで1行取得 (:one は存在しない場合 pgx.ErrNoRows を返す)
SELECT * FROM authors WHERE id = $1;

-- name: ListAuthors :many
-- 全件取得 (:many は []Author を返す)
SELECT * FROM authors ORDER BY created_at;

-- name: UpdateAuthorBio :exec
-- 更新系で戻り値不要な場合は :exec (返り値は error のみ)
UPDATE authors SET bio = $2 WHERE id = $1;

-- name: DeleteAuthor :execrows
-- :execrows は影響を受けた行数を int64 で返す
-- 例: 0 なら対象なし、1 なら削除成功
DELETE FROM authors WHERE id = $1;
