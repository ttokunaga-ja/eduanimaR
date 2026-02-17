# eduanima-librarian（Python 推論マイクロサービス）

本フォルダは eduanima+R における `eduanima-librarian`（知能的専門司書）の **SSOT ドキュメント**。
高速推論モデルを用いて検索戦略と停止判断を行い、Go サーバー（Professor）へ **最小のエビデンス集合**を返す。

## まず読む（最短ルート）
- Docs Portal: [docs/README.md](docs/README.md)
- サービス仕様（SSOT）: [docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md](docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md)
- 責務境界: [docs/01_architecture/MICROSERVICES_MAP.md](docs/01_architecture/MICROSERVICES_MAP.md)
- 依存方向: [docs/01_architecture/CLEAN_ARCHITECTURE.md](docs/01_architecture/CLEAN_ARCHITECTURE.md)
- 技術スタック（SSOT）: [docs/02_tech_stack/STACK.md](docs/02_tech_stack/STACK.md)

## ガードレール（要点）
- Librarian は DB を持たない（DB/バッチ/インデックスは Professor の責務）
- Librarian は最終回答文を生成しない（参照の構造化データのみ）
- 検索ループは `max_retries` を厳守する
