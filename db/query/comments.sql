-- 03-relations で使用するクエリ
-- 示す機能: 1:N 集約 / LEFT JOIN + COUNT / array_agg

-- name: CreateComment :one
INSERT INTO comments (post_id, body)
VALUES ($1, $2)
RETURNING *;

-- name: ListCommentsByPost :many
SELECT * FROM comments WHERE post_id = $1 ORDER BY created_at;

-- name: ListPostsWithCommentCount :many
-- LEFT JOIN + GROUP BY でコメント数を集計する
-- COUNT(c.id) はコメントがない投稿では 0 になる
SELECT
    p.id,
    p.author_id,
    p.title,
    p.body,
    p.tags,
    p.status,
    p.published_at,
    p.created_at,
    COUNT(c.id)::bigint AS comment_count
FROM posts p
LEFT JOIN comments c ON c.post_id = p.id
WHERE p.author_id = $1
GROUP BY p.id
ORDER BY p.created_at;

-- name: ListPostsWithCommentBodies :many
-- array_agg() で 1:N の本文を text[] としてまとめる
-- FILTER (WHERE ...) で NULL を除外し、COALESCE で空配列を保証する
SELECT
    p.id,
    p.title,
    COALESCE(
        array_remove(array_agg(c.body ORDER BY c.created_at), NULL),
        '{}'::text[]
    )::text[] AS comment_bodies
FROM posts p
LEFT JOIN comments c ON c.post_id = p.id
WHERE p.author_id = $1
GROUP BY p.id, p.title
ORDER BY p.created_at;
