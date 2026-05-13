/*
示す sqlc 機能:
  - to_tsvector('simple', ...) @@ plainto_tsquery('simple', ...) — 全文検索
  - ts_rank() で検索スコアを算出して降順ソート
  - 関数式 GIN インデックス (idx_posts_search) が WHERE 節で自動的に使われる
  - 'simple' 辞書: 形態素解析なし、英数字の小文字化のみ。日本語含む多言語に対応。
    本格的な日英形態素解析には 'english' や pg_bigm / pgroonga が必要。
  - sqlc 生成型では ts_rank の戻り値が float32 になる

対応 SQL: db/query/fulltext.sql
マイグレーション: db/migrations/000005_add_posts_search_index.up.sql

実行方法: just run-10  /  go run ./cmd/10-fulltext
DB が起動していない場合は just docker-up && just migrate-up を先に実行すること。
*/
package main

import (
	"context"
	"fmt"
	"log"

	idb "github.com/shnaki/sqlc-example/internal/db"
	"github.com/shnaki/sqlc-example/internal/db/sqlcgen"
)

func main() {
	ctx := context.Background()

	pool, err := idb.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB 接続失敗: %v", err)
	}
	defer pool.Close()

	q := sqlcgen.New(pool)

	fmt.Println("=== 10-fulltext: 全文検索 (tsvector / GIN) ===")
	fmt.Println()

	// セットアップ: 著者を1件作成
	author, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "FullText-Sample",
		Bio:      "全文検索サンプル用著者",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor: %v", err)
	}
	defer q.DeleteAuthor(ctx, author.ID) //nolint:errcheck

	// 検索ヒット用に異なる本文の投稿を4件作成
	posts := []sqlcgen.CreatePostParams{
		{AuthorID: author.ID, Title: "PostgreSQL full text search", Body: "PostgreSQL supports tsvector and GIN indexes for fast search.", Tags: []string{}, Status: sqlcgen.PostStatusPublished},
		{AuthorID: author.ID, Title: "sqlc generates type-safe code", Body: "sqlc reads SQL files and generates Go code from them.", Tags: []string{}, Status: sqlcgen.PostStatusPublished},
		{AuthorID: author.ID, Title: "pgx connection pooling", Body: "pgxpool provides a connection pool for PostgreSQL in Go.", Tags: []string{}, Status: sqlcgen.PostStatusPublished},
		{AuthorID: author.ID, Title: "Go testing with testify", Body: "Use testify assertions to write clear and concise tests in Go.", Tags: []string{}, Status: sqlcgen.PostStatusPublished},
	}
	for _, p := range posts {
		if _, err := q.CreatePost(ctx, p); err != nil {
			log.Fatalf("CreatePost: %v", err)
		}
	}

	// --------------------------------------------------------
	// ヒットするキーワードで検索
	// --------------------------------------------------------
	fmt.Println("--- 検索キーワード: \"postgresql\" ---")
	hits, err := q.SearchPostsFullText(ctx, sqlcgen.SearchPostsFullTextParams{
		PlaintoTsquery: "postgresql",
		Limit:          10,
	})
	if err != nil {
		log.Fatalf("SearchPostsFullText: %v", err)
	}
	fmt.Printf("  ヒット件数: %d\n", len(hits))
	for _, h := range hits {
		fmt.Printf("  - [score=%.4f] %q\n", h.Rank, h.Title)
	}

	// --------------------------------------------------------
	// 複数単語 (AND 検索)
	// --------------------------------------------------------
	fmt.Println("\n--- 検索キーワード: \"go postgresql\" (AND) ---")
	hits2, err := q.SearchPostsFullText(ctx, sqlcgen.SearchPostsFullTextParams{
		PlaintoTsquery: "go postgresql",
		Limit:          10,
	})
	if err != nil {
		log.Fatalf("SearchPostsFullText: %v", err)
	}
	fmt.Printf("  ヒット件数: %d\n", len(hits2))
	for _, h := range hits2 {
		fmt.Printf("  - [score=%.4f] %q\n", h.Rank, h.Title)
	}

	// --------------------------------------------------------
	// ヒットしないキーワード → 空結果
	// --------------------------------------------------------
	fmt.Println("\n--- 検索キーワード: \"nothing-matches\" ---")
	empty, err := q.SearchPostsFullText(ctx, sqlcgen.SearchPostsFullTextParams{
		PlaintoTsquery: "nothing-matches",
		Limit:          10,
	})
	if err != nil {
		log.Fatalf("SearchPostsFullText (empty): %v", err)
	}
	fmt.Printf("  ヒット件数: %d (空結果) ✓\n", len(empty))

	fmt.Println("\n✓ 10-fulltext 完了")
}
