-- 06-advanced で使用するクエリ
-- 示す機能: JSONB / text[] 配列 / CTE + keyset ページネーション / enum 値

-- name: UpsertAuthorMetadata :exec
-- jsonb 列に []byte (JSON) を渡す
-- sqlcgen 側の型: []byte (pgx/v5 のデフォルト)
UPDATE authors
SET metadata = sqlc.arg('metadata')::jsonb
WHERE id = sqlc.arg('id')::uuid;

-- name: GetAuthorMetadata :one
-- jsonb カラムを読み返して []byte として受け取る
SELECT metadata FROM authors WHERE id = $1;

-- name: AddPostTags :exec
-- text[] 列に要素を追加する (array_cat)
-- Go 側の引数 new_tags は []string になる (pgx/v5 + emit_empty_slices)
UPDATE posts
SET tags = array_cat(tags, sqlc.arg('new_tags')::text[])
WHERE id = sqlc.arg('id')::uuid;

-- name: GetPostTags :one
-- text[] を []string として受け取る
SELECT tags FROM posts WHERE id = $1;

-- name: ListPublishedPosts :many
-- PostStatus 型の定数 (PostStatusPublished 等) を使う例
SELECT * FROM posts WHERE status = 'published' ORDER BY created_at DESC;

-- name: KeysetListPosts :many
-- CTE を使った keyset ページネーション
-- cursor_id が NULL の場合は先頭ページ、指定した場合はその投稿の次ページを返す
-- インデックス idx_posts_keyset (created_at DESC, id DESC) を活用する
WITH anchor AS (
    SELECT created_at, id
    FROM posts
    WHERE id = sqlc.narg('cursor_id')::uuid
)
SELECT p.* FROM posts p
WHERE
    sqlc.narg('cursor_id')::uuid IS NULL
    OR (p.created_at, p.id) < (SELECT created_at, id FROM anchor)
ORDER BY p.created_at DESC, p.id DESC
LIMIT sqlc.arg('page_size')::bigint;
