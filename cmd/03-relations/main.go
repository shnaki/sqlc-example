/*
示す sqlc 機能:
  - sqlc.embed() — JOIN クエリで複数テーブルを埋め込み構造体に対応づける
    生成型: GetPostWithAuthorRow { Post Post; Author Author }
    p.* や a.* ではなく sqlc.embed() を使うことでカラム名衝突を回避する
  - LEFT JOIN + COUNT — コメント数を集計して返す
  - array_agg()::text[] — 1:N を配列にまとめて返す

対応 SQL: db/query/posts.sql, db/query/comments.sql

実行方法: just run-03  /  go run ./cmd/03-relations
DB が起動していない場合は just docker-up && just migrate-up を先に実行すること。
*/
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	fmt.Println("=== 03-relations: sqlc.embed / JOIN / array_agg ===\n")

	// セットアップ: 著者・投稿・コメントを作成
	author, err := q.CreateAuthor(ctx, sqlcgen.CreateAuthorParams{
		Name:     "Carol",
		Bio:      "リレーション例の著者",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		log.Fatalf("CreateAuthor: %v", err)
	}
	defer q.DeleteAuthor(ctx, author.ID) //nolint:errcheck

	post1, err := q.CreatePost(ctx, sqlcgen.CreatePostParams{
		AuthorID: author.ID,
		Title:    "sqlc の紹介",
		Body:     "sqlc はタイプセーフな Go コードを生成します",
		Tags:     []string{"sqlc", "go"},
		Status:   sqlcgen.PostStatusPublished,
	})
	if err != nil {
		log.Fatalf("CreatePost1: %v", err)
	}

	post2, err := q.CreatePost(ctx, sqlcgen.CreatePostParams{
		AuthorID: author.ID,
		Title:    "pgx v5 の使い方",
		Body:     "pgx は Go 製 PostgreSQL ドライバです",
		Tags:     []string{"pgx", "go"},
		Status:   sqlcgen.PostStatusDraft,
	})
	if err != nil {
		log.Fatalf("CreatePost2: %v", err)
	}

	// post1 にコメントを追加
	for _, body := range []string{"とても参考になりました!", "sqlc 最高!", "実装してみます"} {
		if _, err := q.CreateComment(ctx, sqlcgen.CreateCommentParams{
			PostID: post1.ID,
			Body:   body,
		}); err != nil {
			log.Fatalf("CreateComment: %v", err)
		}
	}

	// --------------------------------------------------------
	// sqlc.embed() を使った JOIN
	// 生成型: GetPostWithAuthorRow { Post Post; Author Author }
	// --------------------------------------------------------
	fmt.Println("--- GetPostWithAuthor (sqlc.embed) ---")
	row, err := q.GetPostWithAuthor(ctx, post1.ID)
	if err != nil {
		log.Fatalf("GetPostWithAuthor: %v", err)
	}
	fmt.Printf("  投稿: %q (author_id=%s)\n", row.Post.Title, fmtUUID(row.Post.AuthorID))
	fmt.Printf("  著者: %q (id=%s)\n", row.Author.Name, fmtUUID(row.Author.ID))
	fmt.Printf("  → row.Post と row.Author が同一クエリで返ってくる (カラム衝突なし)\n")

	// --------------------------------------------------------
	// LEFT JOIN + COUNT でコメント数を集計
	// --------------------------------------------------------
	fmt.Println("\n--- ListPostsWithCommentCount (LEFT JOIN + COUNT) ---")
	countRows, err := q.ListPostsWithCommentCount(ctx, author.ID)
	if err != nil {
		log.Fatalf("ListPostsWithCommentCount: %v", err)
	}
	for _, r := range countRows {
		fmt.Printf("  投稿 %q: コメント数=%d\n", r.Title, r.CommentCount)
	}

	// --------------------------------------------------------
	// array_agg()::text[] で 1:N を配列にまとめる
	// --------------------------------------------------------
	fmt.Println("\n--- ListPostsWithCommentBodies (array_agg) ---")
	bodyRows, err := q.ListPostsWithCommentBodies(ctx, author.ID)
	if err != nil {
		log.Fatalf("ListPostsWithCommentBodies: %v", err)
	}
	for _, r := range bodyRows {
		if len(r.CommentBodies) == 0 {
			fmt.Printf("  投稿 %q: コメントなし\n", r.Title)
		} else {
			fmt.Printf("  投稿 %q: コメント=[%s]\n", r.Title, strings.Join(r.CommentBodies, " / "))
		}
	}

	// --------------------------------------------------------
	// 1:N を別クエリで取得する方法 (Go 側でアセンブル)
	// --------------------------------------------------------
	fmt.Println("\n--- ListPostsByAuthor + ListCommentsByPost (Go 側でアセンブル) ---")
	posts, err := q.ListPostsByAuthor(ctx, author.ID)
	if err != nil {
		log.Fatalf("ListPostsByAuthor: %v", err)
	}
	for _, p := range posts {
		comments, err := q.ListCommentsByPost(ctx, p.ID)
		if err != nil {
			log.Fatalf("ListCommentsByPost: %v", err)
		}
		fmt.Printf("  投稿 %q: コメント%d件\n", p.Title, len(comments))
	}

	// クリーンアップ (posts/comments は CASCADE DELETE)
	_ = post2.ID

	fmt.Println("\n✓ 03-relations 完了")
}

func fmtUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
