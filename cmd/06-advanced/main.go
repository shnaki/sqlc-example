/*
示す sqlc 機能:
  - JSONB 列   — []byte (json.RawMessage) を渡して PostgreSQL JSONB に保存・取得する
  - text[] 列  — Go の []string を PostgreSQL text[] に読み書きする
  - Enum 値    — PostStatus 型の定数 (PostStatusPublished 等) を使って型安全に絞り込む
  - CTE + keyset ページネーション
      WITH anchor AS (...) + (created_at, id) < $cursor の複合条件で
      ソートと重複なしのページ送りを実現する

対応 SQL: db/query/advanced.sql

実行方法: make run-06  /  go run ./cmd/06-advanced
DB が起動していない場合は make docker-up && make migrate-up を先に実行すること。
*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	idb "github.com/shnaki/sqlc-example/internal/db"
	"github.com/shnaki/sqlc-example/internal/db/sqlcgen"
)

// AuthorProfile は metadata JSONB カラムに格納する Go 構造体。
type AuthorProfile struct {
	GitHubURL string   `json:"github_url"`
	Skills    []string `json:"skills"`
}

func main() {
	ctx := context.Background()

	pool, err := idb.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB 接続失敗: %v", err)
	}
	defer pool.Close()

	q := sqlcgen.New(pool)

	fmt.Println("=== 06-advanced: JSONB / text[] / Enum / CTE keyset ===\n")

	// セットアップ
	author, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Grace-Adv",
		Bio:      "高度機能例の著者",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor: %v", err)
	}
	defer q.DeleteAuthor(ctx, author.ID) //nolint:errcheck

	// ページネーション用に投稿を 8 件作成 (異なる status を混在)
	statuses := []sqlcgen.PostStatus{
		sqlcgen.PostStatusPublished, sqlcgen.PostStatusDraft,
		sqlcgen.PostStatusPublished, sqlcgen.PostStatusArchived,
		sqlcgen.PostStatusPublished, sqlcgen.PostStatusDraft,
		sqlcgen.PostStatusPublished, sqlcgen.PostStatusPublished,
	}
	for i, st := range statuses {
		_, err := q.CreatePost(ctx, sqlcgen.CreatePostParams{
			AuthorID: author.ID,
			Title:    fmt.Sprintf("投稿 #%02d", i+1),
			Body:     "本文",
			Tags:     []string{},
			Status:   st,
		})
		if err != nil {
			log.Fatalf("CreatePost[%d]: %v", i, err)
		}
		// keyset インデックス (created_at DESC, id DESC) が機能するよう少し間を開ける
		time.Sleep(2 * time.Millisecond)
	}

	// --------------------------------------------------------
	// JSONB 列: Go 構造体を JSON に変換して保存・読み返す
	// --------------------------------------------------------
	fmt.Println("--- JSONB 列 (UpsertAuthorMetadata / GetAuthorMetadata) ---")
	profile := AuthorProfile{
		GitHubURL: "https://github.com/grace",
		Skills:    []string{"Go", "PostgreSQL", "sqlc"},
	}
	jsonBytes, err := json.Marshal(profile)
	if err != nil {
		log.Fatalf("json.Marshal: %v", err)
	}

	// []byte を JSONB として保存する
	err = q.UpsertAuthorMetadata(ctx, sqlcgen.UpsertAuthorMetadataParams{
		Metadata: jsonBytes,
		ID:       author.ID,
	})
	if err != nil {
		log.Fatalf("UpsertAuthorMetadata: %v", err)
	}

	// JSONB を []byte として読み返す
	rawMeta, err := q.GetAuthorMetadata(ctx, author.ID)
	if err != nil {
		log.Fatalf("GetAuthorMetadata: %v", err)
	}

	var readBack AuthorProfile
	if err := json.Unmarshal(rawMeta, &readBack); err != nil {
		log.Fatalf("json.Unmarshal: %v", err)
	}
	fmt.Printf("  保存: github=%s skills=%v\n", profile.GitHubURL, profile.Skills)
	fmt.Printf("  読み返し: github=%s skills=%v ✓\n", readBack.GitHubURL, readBack.Skills)

	// --------------------------------------------------------
	// text[] 列: Go の []string を array_cat で追記・取得
	// --------------------------------------------------------
	fmt.Println("\n--- text[] 列 (AddPostTags / GetPostTags) ---")
	// 最初の投稿を取得
	firstPosts, err := q.ListPublishedPosts(ctx)
	if err != nil || len(firstPosts) == 0 {
		log.Fatalf("ListPublishedPosts: %v / len=%d", err, len(firstPosts))
	}
	targetPost := firstPosts[0]

	// [] → ["go", "sqlc"] に追記
	err = q.AddPostTags(ctx, sqlcgen.AddPostTagsParams{
		NewTags: []string{"go", "sqlc"},
		ID:      targetPost.ID,
	})
	if err != nil {
		log.Fatalf("AddPostTags: %v", err)
	}
	// もう一度追記
	err = q.AddPostTags(ctx, sqlcgen.AddPostTagsParams{
		NewTags: []string{"postgresql"},
		ID:      targetPost.ID,
	})
	if err != nil {
		log.Fatalf("AddPostTags2: %v", err)
	}

	tags, err := q.GetPostTags(ctx, targetPost.ID)
	if err != nil {
		log.Fatalf("GetPostTags: %v", err)
	}
	fmt.Printf("  追記後のタグ: [%s]\n", strings.Join(tags, ", "))

	// --------------------------------------------------------
	// Enum 値: PostStatus 型定数で型安全に絞り込む
	// --------------------------------------------------------
	fmt.Println("\n--- Enum 値 (ListPublishedPosts) ---")
	published, err := q.ListPublishedPosts(ctx)
	if err != nil {
		log.Fatalf("ListPublishedPosts: %v", err)
	}
	fmt.Printf("  published 投稿数: %d\n", len(published))
	for _, p := range published {
		// status フィールドは PostStatus 型 — string キャストなしに定数比較できる
		if p.Status == sqlcgen.PostStatusPublished {
			fmt.Printf("  - %q (%s) ✓\n", p.Title, p.Status)
		}
	}

	// --------------------------------------------------------
	// CTE + keyset ページネーション
	// --------------------------------------------------------
	fmt.Println("\n--- CTE + Keyset ページネーション (KeysetListPosts) ---")
	const pageSize = 3

	// 第1ページ: cursor_id = NULL (ゼロ値 pgtype.UUID{Valid:false})
	page1, err := q.KeysetListPosts(ctx, sqlcgen.KeysetListPostsParams{
		CursorID: pgtype.UUID{},   // Valid: false → 先頭ページ
		PageSize:  pageSize,
	})
	if err != nil {
		log.Fatalf("KeysetListPosts(page1): %v", err)
	}
	fmt.Printf("  ページ1 (%d件):\n", len(page1))
	for _, p := range page1 {
		fmt.Printf("    - %q\n", p.Title)
	}

	if len(page1) == pageSize {
		// 最後の投稿の ID を cursor として次ページを取得
		lastPost := page1[len(page1)-1]
		page2, err := q.KeysetListPosts(ctx, sqlcgen.KeysetListPostsParams{
			CursorID: lastPost.ID, // cursor_id = 前ページの末尾
			PageSize:  pageSize,
		})
		if err != nil {
			log.Fatalf("KeysetListPosts(page2): %v", err)
		}
		fmt.Printf("  ページ2 (%d件, cursor=%s):\n", len(page2), fmtUUID(lastPost.ID)[:8]+"...")
		for _, p := range page2 {
			fmt.Printf("    - %q\n", p.Title)
		}
	}

	fmt.Println("\n✓ 06-advanced 完了")
}

func fmtUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
