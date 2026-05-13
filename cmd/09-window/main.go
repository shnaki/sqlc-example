/*
示す sqlc 機能:
  - ROW_NUMBER() OVER (PARTITION BY ... ORDER BY ...) — 著者ごとの新着順番号
    sqlc は window 列を int64 型に推定して生成する
  - LAG(...) OVER (PARTITION BY ... ORDER BY ...) — 1つ前の行の値
    sqlc は戻り型を決定できないため interface{} で生成される
    NULL の場合は nil、値がある場合は string に型アサートする
  - COUNT(*) OVER() — ウィンドウ全体の件数 (LIMIT/OFFSET と組み合わせて総件数+ページを1クエリで取得)

対応 SQL: db/query/window.sql

実行方法: just run-09  /  go run ./cmd/09-window
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

	fmt.Println("=== 09-window: Window 関数 ===")
	fmt.Println()

	// セットアップ: 著者と投稿を用意する
	author, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Window-Sample",
		Bio:      "window 関数サンプル用著者",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor: %v", err)
	}
	defer q.DeleteAuthor(ctx, author.ID) //nolint:errcheck

	titles := []string{
		"投稿 A", "投稿 B", "投稿 C",
		"投稿 D", "投稿 E", "投稿 F",
	}
	for _, t := range titles {
		_, err := q.CreatePost(ctx, sqlcgen.CreatePostParams{
			AuthorID: author.ID,
			Title:    t,
			Body:     t + " の本文",
			Tags:     []string{},
			Status:   sqlcgen.PostStatusDraft,
		})
		if err != nil {
			log.Fatalf("CreatePost(%s): %v", t, err)
		}
	}

	// --------------------------------------------------------
	// ROW_NUMBER: 著者ごとの新着順番号
	// --------------------------------------------------------
	fmt.Println("--- ROW_NUMBER() OVER (PARTITION BY author_id ORDER BY created_at DESC) ---")
	ranked, err := q.ListPostsRankedByAuthor(ctx)
	if err != nil {
		log.Fatalf("ListPostsRankedByAuthor: %v", err)
	}
	// 今回の著者の投稿だけ表示
	for _, r := range ranked {
		if r.AuthorID == author.ID {
			fmt.Printf("  [%d] %s\n", r.RowNum, r.Title)
		}
	}

	// --------------------------------------------------------
	// LAG: 同一著者の1つ前の投稿タイトル
	// --------------------------------------------------------
	fmt.Println("\n--- LAG(title) OVER (PARTITION BY author_id ORDER BY created_at) ---")
	lagged, err := q.ListPostsWithPrevTitle(ctx)
	if err != nil {
		log.Fatalf("ListPostsWithPrevTitle: %v", err)
	}
	for _, r := range lagged {
		if r.AuthorID == author.ID {
			// PrevTitle は interface{} — nil (先頭行) か string (それ以降)
			prev := "(なし)"
			if s, ok := r.PrevTitle.(string); ok {
				prev = s
			}
			fmt.Printf("  %-10s  前の投稿: %s\n", r.Title, prev)
		}
	}

	// --------------------------------------------------------
	// COUNT(*) OVER(): 総件数+ページを1クエリで取得
	// --------------------------------------------------------
	fmt.Println("\n--- COUNT(*) OVER() + LIMIT/OFFSET (ページ2 / 2件ずつ) ---")
	page, err := q.ListPostsPageWithTotal(ctx, sqlcgen.ListPostsPageWithTotalParams{
		Limit:  2,
		Offset: 2,
	})
	if err != nil {
		log.Fatalf("ListPostsPageWithTotal: %v", err)
	}
	if len(page) > 0 {
		fmt.Printf("  全件数: %d 件 (1クエリで取得)\n", page[0].TotalCount)
		for _, r := range page {
			fmt.Printf("  - %s\n", r.Title)
		}
	}

	fmt.Println("\n✓ 09-window 完了")
}
