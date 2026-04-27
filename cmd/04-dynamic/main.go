/*
示す sqlc 機能:
  - sqlc.arg('name')  — パラメータに明示的な名前を付ける (Params 構造体のフィールド名に反映)
  - sqlc.narg('name') — NULL 許容パラメータ (pgtype.UUID{Valid:false} / NullPostStatus{Valid:false} 等)
    IS NULL OR col = $x パターンで動的フィルタを実現する
  - sqlc.slice('name')— pgx/v5 では WHERE id = ANY($1::uuid[]) に展開される ([]pgtype.UUID を渡す)
  - COALESCE + narg   — 部分更新 (NULL を渡した列は元の値を維持)

対応 SQL: db/query/posts_dynamic.sql

実行方法: just run-04  /  go run ./cmd/04-dynamic
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

	fmt.Println("=== 04-dynamic: sqlc.arg / sqlc.narg / sqlc.slice ===\n")

	// セットアップ
	author, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Dave",
		Bio:      "動的クエリ例の著者",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor: %v", err)
	}
	defer q.DeleteAuthor(ctx, author.ID) //nolint:errcheck

	titles := []string{"Go 入門", "sqlc チュートリアル", "PostgreSQL 応用", "Docker 実践"}
	statuses := []sqlcgen.PostStatus{
		sqlcgen.PostStatusPublished,
		sqlcgen.PostStatusPublished,
		sqlcgen.PostStatusDraft,
		sqlcgen.PostStatusArchived,
	}
	var postIDs []pgtype.UUID
	for i, title := range titles {
		p, err := q.CreatePost(ctx, sqlcgen.CreatePostParams{
			AuthorID: author.ID,
			Title:    title,
			Body:     "本文",
			Tags:     []string{},
			Status:   statuses[i],
		})
		if err != nil {
			log.Fatalf("CreatePost: %v", err)
		}
		postIDs = append(postIDs, p.ID)
	}

	// --------------------------------------------------------
	// sqlc.narg: 全フィールドが NULL (ゼロ値) → フィルタなし
	// --------------------------------------------------------
	fmt.Println("--- SearchPosts: フィルタなし (全件) ---")
	all, err := q.SearchPosts(ctx, sqlcgen.SearchPostsParams{
		// AuthorID.Valid == false → フィルタ無効
		// Status.Valid    == false → フィルタ無効
		// TitleLike.Valid == false → フィルタ無効
		LimitCount: 10,
	})
	if err != nil {
		log.Fatalf("SearchPosts(全件): %v", err)
	}
	fmt.Printf("  件数: %d\n", len(all))
	for _, p := range all {
		fmt.Printf("  - %q [%s]\n", p.Title, p.Status)
	}

	// --------------------------------------------------------
	// sqlc.narg: 著者でフィルタ
	// --------------------------------------------------------
	fmt.Println("\n--- SearchPosts: 著者フィルタあり ---")
	byAuthor, err := q.SearchPosts(ctx, sqlcgen.SearchPostsParams{
		AuthorID:   author.ID, // Valid == true → 著者フィルタ有効
		LimitCount: 10,
	})
	if err != nil {
		log.Fatalf("SearchPosts(著者): %v", err)
	}
	fmt.Printf("  件数: %d\n", len(byAuthor))

	// --------------------------------------------------------
	// sqlc.narg: ステータス + タイトル部分一致フィルタ
	// --------------------------------------------------------
	fmt.Println("\n--- SearchPosts: status=published かつ title ILIKE '%sqlc%' ---")
	filtered, err := q.SearchPosts(ctx, sqlcgen.SearchPostsParams{
		AuthorID: author.ID,
		Status: sqlcgen.NullPostStatus{
			PostStatus: sqlcgen.PostStatusPublished,
			Valid:      true,
		},
		TitleLike: pgtype.Text{
			String: "%sqlc%",
			Valid:  true,
		},
		LimitCount: 10,
	})
	if err != nil {
		log.Fatalf("SearchPosts(filtered): %v", err)
	}
	fmt.Printf("  件数: %d\n", len(filtered))
	for _, p := range filtered {
		fmt.Printf("  - %q [%s]\n", p.Title, p.Status)
	}

	// --------------------------------------------------------
	// sqlc.slice: WHERE id = ANY($1::uuid[])
	// IN ($1, $2, ...) ではなく ANY($1::uuid[]) に展開される点に注目
	// --------------------------------------------------------
	fmt.Println("\n--- ListPostsByIDs (sqlc.slice → ANY($1::uuid[])) ---")
	ids := postIDs[:2] // 最初の2件の ID で検索
	byIDs, err := q.ListPostsByIDs(ctx, ids)
	if err != nil {
		log.Fatalf("ListPostsByIDs: %v", err)
	}
	fmt.Printf("  取得件数: %d (指定ID数: %d)\n", len(byIDs), len(ids))
	for _, p := range byIDs {
		fmt.Printf("  - %q\n", p.Title)
	}

	// --------------------------------------------------------
	// sqlc.narg + COALESCE: 部分更新 (title のみ変更、status は維持)
	// --------------------------------------------------------
	fmt.Println("\n--- UpdatePostFlexible (COALESCE 部分更新) ---")
	targetPost := postIDs[0]

	// title だけ更新し、status は NULL を渡して元の値を維持する
	err = q.UpdatePostFlexible(ctx, sqlcgen.UpdatePostFlexibleParams{
		Title:  pgtype.Text{String: "Go 入門 (改訂版)", Valid: true},
		Status: sqlcgen.NullPostStatus{Valid: false}, // NULL → 元の値を維持
		ID:     targetPost,
	})
	if err != nil {
		log.Fatalf("UpdatePostFlexible: %v", err)
	}

	updated, err := q.ListPostsByIDs(ctx, []pgtype.UUID{targetPost})
	if err != nil || len(updated) == 0 {
		log.Fatalf("ListPostsByIDs after update: %v", err)
	}
	fmt.Printf("  更新後: title=%q status=%s (status は変わっていない)\n",
		updated[0].Title, updated[0].Status)

	fmt.Println("\n✓ 04-dynamic 完了")
}
