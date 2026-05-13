-- name: SearchPostsFullText :many
-- to_tsvector('simple', ...) @@ plainto_tsquery('simple', ...) で全文検索。
-- 'simple' 辞書は形態素解析なしで英数字をそのまま正規化する。
-- ts_rank() でスコア降順に並べる。
-- idx_posts_search (GIN) が WHERE 節で使われて高速化される。
SELECT
    id,
    title,
    body,
    ts_rank(
        to_tsvector('simple', title || ' ' || body),
        plainto_tsquery('simple', $1)
    ) AS rank
FROM posts
WHERE to_tsvector('simple', title || ' ' || body)
      @@ plainto_tsquery('simple', $1)
ORDER BY rank DESC
LIMIT $2;
