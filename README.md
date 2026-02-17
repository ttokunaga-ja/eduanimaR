# eduanimaR

学習効果検証AIサービス「eduanimaR」のモノレポ。  
AI（Coding Agent）と人間が迷わずに開発を開始できるよう、設計・運用・契約（SSOT）をドキュメント化したプロジェクト集です。

## サービス構成

| リポジトリ | 役割 | 言語/フレームワーク |
|---|---|---|
| `eduanimaR/` | フロントエンド | Next.js / FSD / TypeScript |
| `eduanimaR_Professor/` | バックエンド（API・DB・検索実行） | Go / Clean Architecture |
| `eduanimaR_Librarian/` | 推論サービス（検索戦略・停止判断） | Python / LangGraph |
| `eduanimaRHandbook/` | サービスコンセプト・戦略・プロダクト定義 | — |

## 読む順（最短）

1. サービス概念: `eduanimaRHandbook/README.md`
2. バックエンド契約（SSOT）: `eduanimaR_Professor/docs/README.md`
3. フロントエンド構成: `eduanimaR/docs/README.md`
4. 推論サービス境界: `eduanimaR_Librarian/docs/README.md`

## 基本方針

- "実装より先にドキュメント（契約）を更新する"
- SSOT（Single Source of Truth）を明示し、推測での実装を避ける
- 例外を増やさず、境界/責務/契約を直して解決する
---

## Phase 1開発開始チェックリスト

このリストは、Phase 1（ローカル開発・Web版完全動作）の実装を開始する前に満たすべき条件です。

### 契約・定義（MUST）
- [x] `eduanimaR_Professor/docs/openapi.yaml`が以下を定義:
  - `POST /v1/auth/dev-login`
  - `POST /v1/subjects/{subject_id}/chats`（SSE）
  - `GET /v1/subjects`
  - `GET /v1/subjects/{subject_id}/materials`
  - `GET /v1/subjects/{subject_id}/chats`
  - `POST /v1/subjects/{subject_id}/chats/{chat_id}/feedback`
- [x] `eduanimaR_Professor/proto/librarian/v1/librarian.proto`が定義済み（Phase 1から使用）
- [x] `eduanimaR_Professor/docs/openapi.librarian.yaml`のgRPC位置づけが明記されている

### バックエンド（Professor）
- [x] `eduanimaR_Professor/docs/01_architecture/DB_SCHEMA_DESIGN.md`にER図・テーブル定義がある
- [ ] `eduanimaR_Professor/docs/05_operations/CI_CD.md`の最低ゲート（lint/test/contract drift）が実装可能
- [ ] `docker-compose.yml`でProfessor + PostgreSQL + Kafka + Librarianが起動できる
- [ ] Professor ↔ Librarian gRPC双方向ストリーミングが実装できる（プロトコル: `proto/librarian/v1/librarian.proto`）
- [ ] Kafka非同期パイプライン（OCR/Embedding）が実装できる

### フロントエンド（eduanimaR Web版）
- [ ] `orval.config.ts`がProfessorの`openapi.yaml`を参照している（`eduanimaR/openapi/openapi.yaml` = Professor SSOT と同期済み）
- [x] `eduanimaR/docs/03_integration/AUTH_SESSION.md`のPhase 1認証スキップ方針が実装可能
- [ ] `http://localhost:8080`でProfessorに接続できる
- [ ] Web版固有機能（科目プルダウン・資料一覧・会話履歴）が実装できる

### Librarian
- [x] `eduanimaR_Librarian/docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`の責務境界が明確
- [x] `eduanimaR_Professor/proto/librarian/v1/librarian.proto`が定義済み（Phase 1から使用）
- [ ] gRPC双方向ストリーミング接続の準備が整っている（実装待ち）

### 開発開始の判断
**契約・定義はすべて完了**しています。残る実装タスク（docker-compose, gRPC/Kafka実装）は開発フェーズで対応します。  
**→ Phase 1 の実装開始が可能です。**

### Phase別スコープ概要

| Phase | ゴール |
|---|---|
| Phase 1 | バックエンド完全版（Kafka/gRPC）+ Web版全機能動作（dev-user固定） |
| Phase 2 | Chrome拡張機能（ZIP配布）+ SSO認証 |
| Phase 3 | Chrome Web Store公開 |
| Phase 4 | 閲覧画面HTML+画像取得→LLM解説（小テスト支援） |
| Phase 5 | 学習計画機能（構想段階） |
