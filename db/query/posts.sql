-- 02-transactions / 03-relations / 05-batch で使用するクエリ

-- name: CreatePost :one
INSERT INTO posts (author_id, title, body, tags, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListPostsByAuthor :many
SELECT * FROM posts WHERE author_id = $1 ORDER BY created_at;

-- name: GetPostWithAuthor :one
-- sqlc.embed() を使った JOIN
-- 生成される行型: GetPostWithAuthorRow { Post Post; Author Author }
-- p.* や a.* ではなく sqlc.embed() を使うことでカラム名衝突を回避する
SELECT sqlc.embed(p), sqlc.embed(a)
FROM posts p
JOIN authors a ON a.id = p.author_id
WHERE p.id = $1;

-- name: BatchInsertPost :batchexec
-- :batchexec → pgx の Batch を使って複数行を一括実行
-- Go 側: q.BatchInsertPost(ctx, []BatchInsertPostParams{...}).Exec(func(i int, err error){})
INSERT INTO posts (author_id, title, body, tags, status)
VALUES ($1, $2, $3, $4, $5);

-- name: BatchGetPost :batchone
-- :batchone → Batch 内で1行を返すクエリ
-- Go 側: .QueryRow(func(i int, p Post, err error){})
SELECT * FROM posts WHERE id = $1;

-- name: BatchListPostsByAuthor :batchmany
-- :batchmany → Batch 内で複数行を返すクエリ
-- Go 側: .Query(func(i int, p Post, err error){})
SELECT * FROM posts WHERE author_id = $1 ORDER BY created_at;
