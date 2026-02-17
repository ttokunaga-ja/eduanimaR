# Quickstart（eduanima-librarian）

目的：Librarian の責務境界・契約・運用の「開始条件」を短時間で確定させる。

## 0) 開始条件（Must）
- サービス境界（Professor ↔ Librarian）が `01_architecture/MICROSERVICES_MAP.md` に反映されている
- Professor ↔ Librarian の契約（gRPC/Proto）が SSOT として場所が決まっている
- CI の最低ゲート（lint/test/contract drift）が `05_operations/CI_CD.md` の方針で組める

## 1) 最短で読む順（推奨）
1. `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
2. `01_architecture/MICROSERVICES_MAP.md`
3. `01_architecture/CLEAN_ARCHITECTURE.md`
4. `02_tech_stack/STACK.md`
5. `03_integration/API_CONTRACT_WORKFLOW.md`
6. `01_architecture/RESILIENCY.md`

## 2) まず埋める（プロジェクト固有）
- `00_quickstart/PROJECT_DECISIONS.md`

---

## Phase 1開始条件（Librarian実装・統合）

### 前提条件

1. **gRPC契約の確認**:
   - `eduanimaR_Professor/proto/librarian/v1/librarian.proto` が定義済み
   - RPC: `Reason(stream ReasoningInput) returns (stream ReasoningOutput)`

2. **Professor側の準備**:
   - Professor が gRPC クライアントとして Librarian へ接続可能
   - Librarian未起動時のフォールバック動作が実装済み（Phase 1での後方互換）

3. **Librarian側の実装**:
   - LangGraph による検索ループの状態管理
   - Gemini 3 Flash による推論（クエリ生成・停止判断）
   - Professor経由での検索実行（DB/GCS直接アクセス禁止）

### Phase 1完了条件

- [ ] gRPC サーバーが起動し、Professor からの接続を受け付ける
- [ ] LangGraph による検索ループが動作する（最大5回試行）
- [ ] Gemini 3 Flash による推論が正常に動作する
- [ ] Professor からの検索要求に対して適切なクエリを生成する
- [ ] 停止条件の満足判定が正しく機能する

---

## Phase 2-5の開始条件

### Phase 2: SSO認証 + 本番環境デプロイ

**開始条件**: Phase 1で実装済みのLibrarianをそのまま本番環境へデプロイ

- [ ] Cloud Run へのデプロイ設定完了
- [ ] Professor から本番環境の Librarian へ gRPC 接続可能

### Phase 3: Chrome Web Store公開

**開始条件**: Phase 2から変更なし（拡張機能のストア公開のみ）

### Phase 4: 閲覧中画面の解説機能追加

**開始条件**: Phase 1で実装済みのLibrarianをそのまま維持

**追加考慮点**:
- 画面HTML・画像解析は Professor側で実施（Gemini Vision API）
- Librarianは従来通りテキストベースの検索クエリ生成のみ

### Phase 5: 学習計画立案機能（構想段階）

**開始条件**（未確定）:
- Phase 1-4の完了
- 学習計画生成のための推論ループ仕様の確定
- 小テスト結果分析のための推論ループ仕様の確定

---

## 参照

- **Phase 1-5詳細**: `../../eduanimaRHandbook/04_product/ROADMAP.md`
- **Professor統合**: `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`
- **gRPC契約**: `../../eduanimaR_Professor/proto/librarian/v1/librarian.proto`
