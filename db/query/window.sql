-- name: ListPostsRankedByAuthor :many
-- ROW_NUMBER() OVER (PARTITION BY ...) で著者ごとの新着順番号を付与する。
-- PARTITION BY で著者ごとにリセット、ORDER BY created_at DESC で新しい投稿が 1 番。
SELECT
    id,
    author_id,
    title,
    created_at,
    ROW_NUMBER() OVER (PARTITION BY author_id ORDER BY created_at DESC) AS row_num
FROM posts
ORDER BY author_id, row_num;

-- name: ListPostsWithPrevTitle :many
-- LAG() OVER (...) で同一著者の1つ前の投稿タイトルを取得する。
-- 先頭行の prev_title は NULL (pgtype.Text{Valid: false}) になる。
SELECT
    id,
    author_id,
    title,
    created_at,
    LAG(title) OVER (PARTITION BY author_id ORDER BY created_at) AS prev_title
FROM posts
ORDER BY author_id, created_at;

-- name: ListPostsPageWithTotal :many
-- COUNT(*) OVER() で全件数を取得しながら LIMIT/OFFSET でページング。
-- 1 クエリで total_count とページ内容を同時に取得できる。
SELECT
    id,
    title,
    created_at,
    COUNT(*) OVER() AS total_count
FROM posts
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;
