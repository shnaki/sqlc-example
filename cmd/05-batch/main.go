/*
示す sqlc 機能:
  - :batchexec  — 複数行を一括実行 (pgx Batch)。コールバック: .Exec(func(i int, err error))
  - :batchone   — Batch 内で1行を返すクエリ。コールバック: .QueryRow(func(i int, row T, err error))
  - :batchmany  — Batch 内で複数行を返すクエリ。コールバック: .Query(func(i int, rows []T, err error))

Batch は複数クエリを1回のネットワーク往復で送信するため、N+1 問題を回避できる。
コールバック内で error を必ず確認すること。Close() は各メソッド内で自動的に呼ばれる。

対応 SQL: db/query/posts.sql (BatchInsertPost / BatchGetPost / BatchListPostsByAuthor)

実行方法: make run-05  /  go run ./cmd/05-batch
DB が起動していない場合は make docker-up && make migrate-up を先に実行すること。
*/
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgtype"
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

	fmt.Println("=== 05-batch: :batchexec / :batchone / :batchmany ===\n")

	// セットアップ: 著者を2人作る
	author1, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Eve-Batch",
		Bio:      "バッチ例の著者1",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor1: %v", err)
	}
	defer q.DeleteAuthor(ctx, author1.ID) //nolint:errcheck

	author2, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Frank-Batch",
		Bio:      "バッチ例の著者2",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor2: %v", err)
	}
	defer q.DeleteAuthor(ctx, author2.ID) //nolint:errcheck

	// --------------------------------------------------------
	// :batchexec — 複数の INSERT を1往復で実行
	// --------------------------------------------------------
	fmt.Println("--- BatchInsertPost (:batchexec) ---")
	params := []sqlcgen.BatchInsertPostParams{
		{AuthorID: author1.ID, Title: "Batch-Post-1", Body: "本文1", Tags: []string{"go"}, Status: sqlcgen.PostStatusPublished},
		{AuthorID: author1.ID, Title: "Batch-Post-2", Body: "本文2", Tags: []string{"sqlc"}, Status: sqlcgen.PostStatusDraft},
		{AuthorID: author2.ID, Title: "Batch-Post-3", Body: "本文3", Tags: []string{"pgx"}, Status: sqlcgen.PostStatusPublished},
		{AuthorID: author2.ID, Title: "Batch-Post-4", Body: "本文4", Tags: []string{}, Status: sqlcgen.PostStatusArchived},
	}

	insertErrors := 0
	// SendBatch で全クエリを一括送信し、Exec コールバックで各結果を受け取る
	q.BatchInsertPost(ctx, params).Exec(func(i int, err error) {
		if err != nil {
			fmt.Printf("  [%d] エラー: %v\n", i, err)
			insertErrors++
		} else {
			fmt.Printf("  [%d] 挿入成功: %s\n", i, params[i].Title)
		}
	})
	if insertErrors > 0 {
		log.Fatalf("BatchInsertPost: %d 件エラー", insertErrors)
	}
	fmt.Printf("  合計 %d 件を1往復で挿入\n", len(params))

	// --------------------------------------------------------
	// :batchone — ID リストで各投稿を1往復で取得
	// --------------------------------------------------------
	fmt.Println("\n--- BatchGetPost (:batchone) ---")
	posts1, err := q.ListPostsByAuthor(ctx, author1.ID)
	if err != nil {
		log.Fatalf("ListPostsByAuthor: %v", err)
	}

	ids := make([]pgtype.UUID, len(posts1))
	for i, p := range posts1 {
		ids[i] = p.ID
	}

	// QueryRow コールバックで各 ID に対応する Post を受け取る
	q.BatchGetPost(ctx, ids).QueryRow(func(i int, p sqlcgen.Post, err error) {
		if err != nil {
			fmt.Printf("  [%d] エラー: %v\n", i, err)
			return
		}
		fmt.Printf("  [%d] 取得: %q [%s]\n", i, p.Title, p.Status)
	})

	// --------------------------------------------------------
	// :batchmany — 著者ごとの投稿リストを1往復で取得
	// --------------------------------------------------------
	fmt.Println("\n--- BatchListPostsByAuthor (:batchmany) ---")
	authorIDs := []pgtype.UUID{author1.ID, author2.ID}

	// Query コールバックで各著者の投稿スライスを受け取る
	q.BatchListPostsByAuthor(ctx, authorIDs).Query(func(i int, posts []sqlcgen.Post, err error) {
		if err != nil {
			fmt.Printf("  [著者%d] エラー: %v\n", i, err)
			return
		}
		fmt.Printf("  [著者%d] 投稿数: %d\n", i, len(posts))
		for _, p := range posts {
			fmt.Printf("    - %q [%s]\n", p.Title, p.Status)
		}
	})

	fmt.Println("\n✓ 05-batch 完了")
}
