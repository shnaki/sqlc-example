/*
示す sqlc 機能:
  - :one   — INSERT/SELECT で1行を返す (Author 型)
  - :many  — SELECT で複数行を返す ([]Author 型)
  - :exec  — UPDATE など戻り値不要な変更操作 (error のみ返す)
  - :execrows — DELETE など影響行数を返す (int64 返す)

対応 SQL: db/query/authors.sql

実行方法: just run-01  /  go run ./cmd/01-basics
DB が起動していない場合は just docker-up && just migrate-up を先に実行すること。
*/
package main

import (
	"context"
	"encoding/json"
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

	fmt.Println("=== 01-basics: :one / :many / :exec / :execrows ===\n")

	// --------------------------------------------------------
	// :one — INSERT ... RETURNING * で Author 構造体を返す
	// --------------------------------------------------------
	fmt.Println("--- CreateAuthor (:one) ---")
	meta, _ := json.Marshal(map[string]string{"lang": "Go"})
	author, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Alice",
		Bio:      "Go と PostgreSQL が好きな開発者",
		Metadata: meta,
	})
	if err != nil {
		log.Fatalf("CreateAuthor: %v", err)
	}
	fmt.Printf("  作成: id=%s name=%s bio=%s\n", fmtUUID(author.ID), author.Name, author.Bio)

	// --------------------------------------------------------
	// :one — 主キーで1行取得 (存在しない場合は pgx.ErrNoRows)
	// --------------------------------------------------------
	fmt.Println("\n--- GetAuthor (:one) ---")
	got, err := q.GetAuthor(ctx, author.ID)
	if err != nil {
		log.Fatalf("GetAuthor: %v", err)
	}
	fmt.Printf("  取得: id=%s name=%s\n", fmtUUID(got.ID), got.Name)

	// 2人目を作って :many の動作を確認する
	author2, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Bob",
		Bio:      "Rust と Go が好きな開発者",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor2: %v", err)
	}

	// --------------------------------------------------------
	// :many — 全件取得で []Author が返る
	// --------------------------------------------------------
	fmt.Println("\n--- ListAuthors (:many) ---")
	authors, err := q.ListAuthors(ctx)
	if err != nil {
		log.Fatalf("ListAuthors: %v", err)
	}
	fmt.Printf("  取得件数: %d\n", len(authors))
	for _, a := range authors {
		fmt.Printf("  - %s: %s\n", a.Name, a.Bio)
	}

	// --------------------------------------------------------
	// :exec — 更新系で返り値不要な場合 (error のみ)
	// --------------------------------------------------------
	fmt.Println("\n--- UpdateAuthorBio (:exec) ---")
	err = q.UpdateAuthorBio(ctx, sqlcgen.UpdateAuthorBioParams{
		ID:  author.ID,
		Bio: "Go, PostgreSQL, sqlc を愛する開発者",
	})
	if err != nil {
		log.Fatalf("UpdateAuthorBio: %v", err)
	}
	updated, _ := q.GetAuthor(ctx, author.ID)
	fmt.Printf("  更新後 bio: %s\n", updated.Bio)

	// --------------------------------------------------------
	// :execrows — 影響行数を int64 で返す
	// --------------------------------------------------------
	fmt.Println("\n--- DeleteAuthor (:execrows) ---")
	n, err := q.DeleteAuthor(ctx, author.ID)
	if err != nil {
		log.Fatalf("DeleteAuthor: %v", err)
	}
	fmt.Printf("  削除行数: %d\n", n)

	// 存在しない ID を削除すると 0 が返る
	n2, _ := q.DeleteAuthor(ctx, pgtype.UUID{}) // Valid=false → NULL の UUID は存在しない
	fmt.Printf("  存在しない ID を削除: 削除行数=%d (0 になる)\n", n2)

	// クリーンアップ
	q.DeleteAuthor(ctx, author2.ID) //nolint:errcheck

	fmt.Println("\n✓ 01-basics 完了")
}

// fmtUUID は pgtype.UUID を "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" 形式の文字列に変換する。
func fmtUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
