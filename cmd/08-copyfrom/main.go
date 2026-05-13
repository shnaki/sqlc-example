/*
示す sqlc 機能:
  - :copyfrom アノテーション — pgx の CopyFrom プロトコルを使った高速バルク INSERT
    生成シグネチャ: func (q *Queries) CopyInsertPosts(ctx, []CopyInsertPostsParams) (int64, error)
  - :batchexec との違い:
    :batchexec  → 複数の INSERT 文を1回のラウンドトリップにまとめる (Extended Query プロトコル)
    :copyfrom   → PostgreSQL COPY バイナリプロトコル — さらに高速、1000件超のバルク挿入に適する

対応 SQL: db/query/copyfrom.sql

実行方法: just run-08  /  go run ./cmd/08-copyfrom
DB が起動していない場合は just docker-up && just migrate-up を先に実行すること。
*/
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shnaki/sqlc-example/internal/db/sqlcgen"

	idb "github.com/shnaki/sqlc-example/internal/db"
)

const batchSize = 1000

func main() {
	ctx := context.Background()

	pool, err := idb.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB 接続失敗: %v", err)
	}
	defer pool.Close()

	q := sqlcgen.New(pool)

	fmt.Println("=== 08-copyfrom: :copyfrom によるバルク INSERT ===")
	fmt.Println()

	// セットアップ: 著者を1件作成
	author, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "CopyFrom-Sample",
		Bio:      "copyfrom サンプル用著者",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor: %v", err)
	}
	// authors を消せば posts も CASCADE で消える
	defer q.DeleteAuthor(ctx, author.ID) //nolint:errcheck

	// --------------------------------------------------------
	// 1000件分のパラメータを組み立て
	// --------------------------------------------------------
	params := make([]sqlcgen.CopyInsertPostsParams, batchSize)
	for i := range params {
		params[i] = sqlcgen.CopyInsertPostsParams{
			AuthorID: author.ID,
			Title:    fmt.Sprintf("CopyFrom Post #%04d", i+1),
			Body:     fmt.Sprintf("本文 %d", i+1),
			Tags:     []string{"copyfrom", "bulk"},
			Status:   sqlcgen.PostStatusDraft,
		}
	}

	// --------------------------------------------------------
	// CopyInsertPosts: 1回の呼び出しで 1000件を一括 INSERT
	// --------------------------------------------------------
	fmt.Printf("  %d 件を CopyFrom で INSERT 中...\n", batchSize)
	start := time.Now()

	inserted, err := q.CopyInsertPosts(ctx, params)
	if err != nil {
		log.Fatalf("CopyInsertPosts: %v", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("  挿入件数: %d 件\n", inserted)
	fmt.Printf("  所要時間: %v\n", elapsed)
	fmt.Printf("  スループット: %.0f 件/秒\n", float64(inserted)/elapsed.Seconds())

	fmt.Println("\n✓ 08-copyfrom 完了")
}
