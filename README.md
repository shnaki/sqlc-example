# sqlc-example — Go sqlc 学習用カタログ

sqlc の主要機能をテーマ別に分けた、Go の小さな実行可能サンプル集。  
Web API ではなく、各 `cmd/<NN>-<topic>/main.go` を **読んで実行する**学習用カタログ。

## このリポジトリの特徴

- **テーマ単位で独立**: 各 `cmd/` は単独で `go run` できる
- **SQL → 生成物 → Go の流れ**: `db/query/*.sql` → `make sqlc` → `internal/db/sqlcgen/` → `cmd/*/main.go`
- **studytrack-api と同じ規約**: pgx v5 / golang-migrate / `tool` ディレクティブ / Makefile

## 動作環境

- Go 1.26 以上
- Docker (PostgreSQL 18 を起動するため)

## セットアップ手順

```bash
cp .env.example .env          # DB_URL を確認 (デフォルトで動作する)
make docker-up                # postgres:18-alpine をホスト :5437 で起動
# healthy になるまで少し待つ
make migrate-up               # 全マイグレーション適用
make sqlc                     # クエリを再生成 (DB が不要)
make build                    # 全サンプルのビルド確認
```

## サンプル一覧

| 番号 | テーマ | 示す sqlc 機能 |
|------|--------|--------------|
| 01 | basics       | `:one` / `:many` / `:exec` / `:execrows` |
| 02 | transactions | `Queries.WithTx(tx)` + `defer tx.Rollback()` パターン |
| 03 | relations    | `sqlc.embed()` で JOIN / `array_agg` で 1:N 集約 |
| 04 | dynamic      | `sqlc.arg` / `sqlc.narg` / `sqlc.slice` で動的クエリ |
| 05 | batch        | `:batchexec` / `:batchone` / `:batchmany` |
| 06 | advanced     | JSONB `[]byte` / `text[]` `[]string` / Enum 定数 / CTE keyset ページネーション |

## 実行コマンド

```bash
make run-01   # = go run ./cmd/01-basics
make run-02   # = go run ./cmd/02-transactions
make run-03   # = go run ./cmd/03-relations
make run-04   # = go run ./cmd/04-dynamic
make run-05   # = go run ./cmd/05-batch
make run-06   # = go run ./cmd/06-advanced
```

実行前に DB が起動していることを確認する (`make docker-up && make migrate-up`)。

## 各テーマの読み方

```
1. db/query/<topic>.sql        # どんな SQL を書いたか
2. internal/db/sqlcgen/        # sqlc が何を生成したか (型・関数名)
3. cmd/<NN>-<topic>/main.go    # 生成型をどう使うか (冒頭コメントに学習ポイント)
```

## 開発コマンド

```bash
make fmt           # goimports でフォーマット
make lint          # golangci-lint
make test          # go test ./...
make build         # go build ./...
make sqlc          # sqlc generate (SQL 変更後に実行)
make migrate-up    # 未適用マイグレーションをすべて適用
make migrate-down  # 最新の1ステップを戻す
make docker-up     # postgres:18-alpine を起動
make docker-down   # コンテナを停止
```

## トラブルシューティング

**ポート 5437 が衝突する**  
`docker-compose.yml` と `.env` の `5437` を空きポートに変更する。

**`docker compose up` で PostgreSQL 18 のデータディレクトリエラーが出る**  
古い `sqlc-example_sqlcdemo-pgdata` ボリュームが残っている可能性がある。  
`docker volume rm sqlc-example_sqlcdemo-pgdata` を実行してから `make docker-up` をやり直す。

**`make sqlc` が失敗する**  
`sqlc.yaml` の `engine` / `sql_package` / `out` のパスを確認する。  
sqlc は DB 接続なしに SQL ファイルのみから生成する。

**`pgtype.UUID` の扱い**  
DB から返る UUID は `pgtype.UUID{Bytes: [16]byte{...}, Valid: true}`。  
`Valid: false` (ゼロ値) は SQL の NULL に対応する。

**golang-migrate の DSN**  
`cmd/migrate/main.go` は `postgres://...` 形式の DSN を使用する (pgx5:// ではない)。

## 発展課題 (README を読み終えたら)

- **type override**: `sqlc.yaml` の `overrides` で `uuid` → `github.com/google/uuid` に変換する
- **emit_prepared_queries**: プリペアドステートメントを有効化する (true にして再生成)
- **`Querier` interface のモック**: `emit_interface: true` で生成された `Querier` を使ってテストを書く
- **enum の追加**: `000004_add_status.up.sql` でステータスを追加し `make sqlc` でどう変わるか確認する
