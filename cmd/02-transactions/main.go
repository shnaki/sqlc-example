/*
示す sqlc 機能:
  - Queries.WithTx(tx) — pgx のトランザクションを sqlc クエリに渡す
  - defer tx.Rollback(ctx) イディオム — Commit 後の Rollback は no-op になる
  - 成功パス: CreateAuthor + CreatePost を同一 Tx 内で実行して Commit
  - 失敗パス: 途中で Rollback し、変更が巻き戻ることを確認

対応 SQL: db/query/authors.sql, db/query/posts.sql

実行方法: make run-02  /  go run ./cmd/02-transactions
DB が起動していない場合は make docker-up && make migrate-up を先に実行すること。
*/
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
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

	fmt.Println("=== 02-transactions: Queries.WithTx(tx) ===\n")

	// --------------------------------------------------------
	// 成功パス: トランザクション内で2つの書き込みをアトミックに実行
	// --------------------------------------------------------
	fmt.Println("--- 成功パス: Commit ---")

	beforeCount := mustCount(ctx, q)
	fmt.Printf("  コミット前の著者数: %d\n", beforeCount)

	// トランザクション開始
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Fatalf("BeginTx: %v", err)
	}
	// Commit 後に呼ばれる Rollback は no-op になる (failsafe パターン)
	defer tx.Rollback(ctx) //nolint:errcheck

	// WithTx(tx) でトランザクション内のクエリを実行する
	qtx := q.WithTx(tx)

	author, err := qtx.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "TX-Author",
		Bio:      "トランザクション内で作成",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor in Tx: %v", err)
	}
	fmt.Printf("  Tx 内で著者を作成: %s\n", author.Name)

	post, err := qtx.CreatePost(ctx, sqlcgen.CreatePostParams{
		AuthorID: author.ID,
		Title:    "TX-Post",
		Body:     "これはトランザクション内で作成された投稿",
		Tags:     []string{"tx", "sqlc"},
		Status:   sqlcgen.PostStatusDraft,
	})
	if err != nil {
		log.Fatalf("CreatePost in Tx: %v", err)
	}
	fmt.Printf("  Tx 内で投稿を作成: %s\n", post.Title)

	// Commit — これ以降は defer の Rollback は no-op になる
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Commit: %v", err)
	}
	fmt.Println("  Commit 成功 ✓")

	afterCount := mustCount(ctx, q)
	fmt.Printf("  コミット後の著者数: %d (+%d)\n", afterCount, afterCount-beforeCount)

	// --------------------------------------------------------
	// 失敗パス: Rollback でトランザクション内の変更を巻き戻す
	// --------------------------------------------------------
	fmt.Println("\n--- 失敗パス: Rollback ---")

	beforeCount2 := mustCount(ctx, q)
	fmt.Printf("  ロールバック前の著者数: %d\n", beforeCount2)

	tx2, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Fatalf("BeginTx2: %v", err)
	}

	qtx2 := q.WithTx(tx2)
	author2, err := qtx2.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Rollback-Author",
		Bio:      "このレコードはロールバックされる",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		tx2.Rollback(ctx) //nolint:errcheck
		log.Fatalf("CreateAuthor in Tx2: %v", err)
	}
	fmt.Printf("  Tx 内で著者を作成 (まだ見えない): %s\n", author2.Name)

	// 意図的にロールバック — Commit を呼ばずに終了
	if err := tx2.Rollback(ctx); err != nil {
		log.Printf("  Rollback: %v", err)
	}
	fmt.Println("  Rollback 実行 ✓")

	afterCount2 := mustCount(ctx, q)
	fmt.Printf("  ロールバック後の著者数: %d (変化なし: %v)\n",
		afterCount2, afterCount2 == beforeCount2)

	// クリーンアップ
	cleanup(ctx, q, author.ID, post.ID)

	fmt.Println("\n✓ 02-transactions 完了")
}

func mustCount(ctx context.Context, q *sqlcgen.Queries) int {
	authors, err := q.ListAuthors(ctx)
	if err != nil {
		log.Fatalf("ListAuthors: %v", err)
	}
	return len(authors)
}

func cleanup(ctx context.Context, q *sqlcgen.Queries, authorID, _ pgtype.UUID) {
	// posts は CASCADE DELETE で自動的に消える
	q.DeleteAuthor(ctx, authorID) //nolint:errcheck
}

// fmtUUID は pgtype.UUID を文字列に変換する。
func fmtUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
