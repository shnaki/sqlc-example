# 開発ガイドライン

## 目的

本リポジトリは **sqlc の主要機能を学ぶための学習用カタログ**。  
Web API ではなく、`cmd/<NN>-<topic>/main.go` ごとに独立した小さな実行可能プログラムを提供する。  
各ファイルを読み、`go run` で動かすことで、sqlc の機能をひとつずつ最小コードで理解できる。

## 採用技術

| 役割         | ライブラリ                          |
|------------|-------------------------------|
| Go バージョン   | 1.26+                         |
| DB         | PostgreSQL 18 (Docker Compose) |
| DBドライバ     | pgx v5                        |
| コード生成      | sqlc                          |
| マイグレーション   | golang-migrate                |
| フォーマット     | goimports                     |
| 静的解析       | golangci-lint                 |

すべてのツールは go.mod の `tool` ディレクティブ経由で管理する (`go tool sqlc` 等)。

## ディレクトリ規約

```
cmd/<NN>-<topic>/main.go   # 各サンプル (独立して実行可能)
cmd/migrate/main.go        # マイグレーションランナー
db/migrations/             # golang-migrate 用の SQL ファイル (up/down ペア)
db/query/                  # sqlc 用のクエリファイル
internal/db/conn.go        # pgxpool ヘルパ
internal/db/sqlcgen/       # sqlc 生成物 (手書き禁止)
```

## sqlc 利用ルール

- `internal/db/sqlcgen/` は **手で書かない** — 必ず `make sqlc` で生成する。
- `cmd/*/main.go` は SQL を直接書かない — すべて `db/query/*.sql` に書いて `make sqlc` を実行する。
- `pgxpool.Pool` はプログラム起動時に1度だけ生成し、都度 `New()` しない。
- 新しいクエリを追加する場合の手順:
  1. `db/query/*.sql` に SQL を追加
  2. `make sqlc` で生成物を更新
  3. `cmd/` 側から生成型を使って実装

## マイグレーション規約

- `make migrate-create name=xxx` でファイルを生成し、up/down を必ずペアで書く。
- マイグレーションファイルの順序: 依存先テーブルを先に作る (authors → posts → comments)。
- `enum` 型は最初に使うテーブルと同じファイルに置き、down で `DROP TYPE` も書く。

## テーマ追加ルール

新しい sqlc 機能を示すサンプルを追加する場合:

1. `cmd/0N-<topic>/main.go` を新設
2. 必要なら `db/query/xxx.sql` を追加し `make sqlc` を実行
3. `README.md` のサンプル一覧テーブルに行を追加

## アンチパターン

- `internal/db/sqlcgen` を直接編集する
- `cmd/*/main.go` の中に SQL 文字列を書く
- `pgxpool.Pool` をクエリ実行のたびに `New()` する
- `pgx.ErrNoRows` を無視して空の構造体を返す

## コーディング規約

- `make fmt` (`goimports`) 必須
- `make lint` (golangci-lint) を通す
- Conventional Commits 形式でコミットを作成し、説明は日本語で記述する

## コミット規約

`<type>: <説明>` の形式で記述。使用可能なプリフィックス:

- `feat`: 新機能
- `fix`: バグ修正
- `docs`: ドキュメントのみの変更
- `chore`: ビルド・ツール・設定に関する変更
- `refactor`: 機能変更を伴わないコード整理
