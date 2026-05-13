/*
示す sqlc 機能:
  - INSERT ... ON CONFLICT ... DO UPDATE — 真の UPSERT パターン
    EXCLUDED.* で「挿入しようとした値」を参照して UPDATE に使う
  - INSERT ... ON CONFLICT ... DO NOTHING — 衝突時に何もしない
    戻り値 (int64) が 0 = 衝突あり、1 = 新規挿入
  - UNIQUE 制約 (authors_email_unique) を conflict target に指定する
    PostgreSQL の UNIQUE 制約は NULL を「値なし」として扱うため、
    複数の NULL 行は衝突しない (NULL != NULL)

対応 SQL: db/query/upsert.sql
マイグレーション: db/migrations/000004_add_authors_email.up.sql

実行方法: just run-07  /  go run ./cmd/07-upsert
DB が起動していない場合は just docker-up && just migrate-up を先に実行すること。
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

	fmt.Println("=== 07-upsert: INSERT ... ON CONFLICT ===")
	fmt.Println()

	email := pgtype.Text{String: "upsert-sample@example.com", Valid: true}

	// --------------------------------------------------------
	// 1回目: 新規挿入
	// --------------------------------------------------------
	fmt.Println("--- 1回目: 新規挿入 ---")
	first, err := q.UpsertAuthorByEmail(ctx, sqlcgen.UpsertAuthorByEmailParams{
		Name:     "Alice (初回)",
		Bio:      "最初の登録",
		Metadata: []byte(`{}`),
		Email:    email,
	})
	if err != nil {
		log.Fatalf("UpsertAuthorByEmail (1回目): %v", err)
	}
	defer q.DeleteAuthor(ctx, first.ID) //nolint:errcheck

	fmt.Printf("  ID:   %s\n", fmtUUID(first.ID))
	fmt.Printf("  Name: %s\n", first.Name)
	fmt.Printf("  Bio:  %s\n", first.Bio)

	// --------------------------------------------------------
	// 2回目: 同一 email → UPDATE (ID は変わらない)
	// --------------------------------------------------------
	fmt.Println("\n--- 2回目: 同一 email → ON CONFLICT DO UPDATE ---")
	second, err := q.UpsertAuthorByEmail(ctx, sqlcgen.UpsertAuthorByEmailParams{
		Name:     "Alice (更新後)",
		Bio:      "bio を上書きした",
		Metadata: []byte(`{"updated": true}`),
		Email:    email,
	})
	if err != nil {
		log.Fatalf("UpsertAuthorByEmail (2回目): %v", err)
	}

	fmt.Printf("  ID:   %s (1回目と同じ: %v)\n", fmtUUID(second.ID), second.ID == first.ID)
	fmt.Printf("  Name: %s\n", second.Name)
	fmt.Printf("  Bio:  %s\n", second.Bio)

	// --------------------------------------------------------
	// DO NOTHING: 衝突しても INSERT しない → 戻り行数 0
	// --------------------------------------------------------
	fmt.Println("\n--- DO NOTHING: 衝突時は何もしない ---")
	rowsInserted, err := q.InsertAuthorIgnoreConflict(ctx, sqlcgen.InsertAuthorIgnoreConflictParams{
		Name:     "Bob",
		Bio:      "衝突するはずの行",
		Metadata: []byte(`{}`),
		Email:    email, // Alice と同じ email
	})
	if err != nil {
		log.Fatalf("InsertAuthorIgnoreConflict: %v", err)
	}
	fmt.Printf("  挿入件数: %d (0 = 衝突して何もしなかった) ✓\n", rowsInserted)

	// --------------------------------------------------------
	// DO NOTHING: email = NULL → 部分インデックスが適用されないため衝突しない
	// --------------------------------------------------------
	fmt.Println("\n--- DO NOTHING: email = NULL → 衝突なし ---")
	noEmail := pgtype.Text{Valid: false}
	rowsA, err := q.InsertAuthorIgnoreConflict(ctx, sqlcgen.InsertAuthorIgnoreConflictParams{
		Name:     "Charlie (email なし)",
		Bio:      "email NULL は衝突しない",
		Metadata: []byte(`{}`),
		Email:    noEmail,
	})
	if err != nil {
		log.Fatalf("InsertAuthorIgnoreConflict (null email 1回目): %v", err)
	}
	rowsB, err := q.InsertAuthorIgnoreConflict(ctx, sqlcgen.InsertAuthorIgnoreConflictParams{
		Name:     "Dave (email なし)",
		Bio:      "同じく email NULL — 衝突しない",
		Metadata: []byte(`{}`),
		Email:    noEmail,
	})
	if err != nil {
		log.Fatalf("InsertAuthorIgnoreConflict (null email 2回目): %v", err)
	}
	fmt.Printf("  1件目挿入: %d, 2件目挿入: %d (NULL 同士は競合しない) ✓\n", rowsA, rowsB)

	fmt.Println("\n✓ 07-upsert 完了")
}

func fmtUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
