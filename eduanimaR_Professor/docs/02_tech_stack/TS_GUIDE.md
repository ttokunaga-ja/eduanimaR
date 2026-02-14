# TS_GUIDE

## 位置づけ
本リポジトリはバックエンド（Professor: Go）中心だが、契約（OpenAPI/Proto）や周辺ツールの都合で TypeScript を使うケースがある。
このテンプレは以下どちらでも運用できる:
- **同一リポジトリ運用**: FE/BE を同居させ、FSD/Next.js のルールを適用
- **分離リポジトリ運用**: 本リポジトリは BE/運用/契約の SSOT とし、FE は別リポジトリで運用

TypeScript は原則「周辺ツール用途」（例: OpenAPIからの型生成、ドキュメント整形など）に限定する。

## ルール
- アプリ本体の実装言語は Go を正とする
- TS導入が必要な場合は、目的（何を自動化するか）と運用（CI実行、依存更新）を明文化する

## eduanima+R 固有の前提
- Frontend が直接呼ぶのは Professor の OpenAPI（HTTP/JSON）と SSE
- Librarian は gRPC で Professor のみが呼ぶ（Frontendから直接は呼ばない）

