-- 04-dynamic で使用するクエリ
-- 示す機能: sqlc.arg() / sqlc.narg() / sqlc.slice()

-- name: SearchPosts :many
-- sqlc.narg('x') → NULL 許容引数を生成する (NullXxx 型)
-- IS NULL OR col = $x パターンで動的フィルタを実現する
SELECT * FROM posts
WHERE
    (sqlc.narg('author_id')::uuid IS NULL OR author_id = sqlc.narg('author_id')::uuid)
    AND (sqlc.narg('status')::post_status IS NULL OR status = sqlc.narg('status')::post_status)
    AND (sqlc.narg('title_like')::text IS NULL OR title ILIKE sqlc.narg('title_like')::text)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit_count')::bigint;

-- name: ListPostsByIDs :many
-- sqlc.slice('x') → pgx/v5 では WHERE id = ANY($1::uuid[]) に展開される
-- Go 側の引数は []pgtype.UUID になる (IN ($1,$2,...) ではない点に注意)
SELECT * FROM posts WHERE id = ANY(sqlc.slice('ids')::uuid[]);

-- name: UpdatePostFlexible :exec
-- sqlc.narg() + COALESCE パターンで部分更新を実現する
-- NULL を渡した列は元の値を維持する
UPDATE posts
SET
    title  = COALESCE(sqlc.narg('title')::text, title),
    status = COALESCE(sqlc.narg('status')::post_status, status)
WHERE id = sqlc.arg('id')::uuid;
