# STACK（eduanima-librarian SSOT）

## サマリ
`eduanima-librarian` は **Python のステートレス推論マイクロサービス**。
Gemini 3 Flash を用いて検索戦略・停止判断・エビデンス選定を行い、検索の物理実行は Go 側（Professor）へ委譲する。

## 技術スタック（2026-02-14 時点の方針）

| 項目 | 採用 | 備考 |
| :--- | :--- | :--- |
| **Runtime** | Python（ASGI） | 具体バージョンはプロジェクトで pin する（例: 3.12+）。本サービスは DB を持たない。 |
| **Web Framework** | Litestar | gRPCサーバー実装（`/v1/librarian/search-agent`）。 |
| **Schema/Serialization** | msgspec / Protocol Buffers | Professor ↔ Librarian の契約（DTO）を高速・厳格に扱う。gRPC契約は Protocol Buffers。 |
| **Agent Orchestration** | LangGraph | 検索ループの状態管理、MaxRetry/停止条件の保証。 |
| **HTTP Client** | （例: httpx） | Professor の検索ツール呼び出し（gRPC経由）、および Gemini API 呼び出しに使用。 |
| **LLM** | Gemini 3 Flash | Librarian の標準推論モデル（戦略立案/停止判断/選定）。 |
| **Observability** | OpenTelemetry（Python） | trace/log correlation を前提にする（request_id/trace_id）。 |
| **Packaging/Build** | pyproject ベース | 依存は lock して再現性を確保（ツールはプロジェクトで固定）。 |

## Gemini 3 Flash の使い分け（推奨）
- 戦略立案（Plan）: 思考コストを許容（例: medium 相当）
- ループ中の微修正（Refine/Evaluate）: 低コスト（例: low 相当）

> 具体のパラメータ名/SDKは採用SDKに依存するため、実装側で SSOT 化する。

## SSOT（Single Source of Truth）
- Librarian の仕様: `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
- 責務境界: `01_architecture/MICROSERVICES_MAP.md`
- gRPC 契約: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`

## 明確に「やらない」こと
- DB/インデックス/バッチ処理（Professor の責務）
- gRPC 以外の内部 RPC 方式の独自採用（Professor との契約は gRPC/Proto が正）

## Phase 1-5での取り扱い

### Phase 1: バックエンド完成（ローカル開発）

**Librarian責務**:
- Phase 3の小戦略実行: クエリ生成（最大5回試行）
- 停止条件の満足判定
- Professor経由での検索実行（DB/GCS直接アクセス禁止）
- LangGraphによる検索ループの状態管理

**統合方法**:
- gRPC契約（`eduanimaR_Professor/proto/librarian/v1/librarian.proto`）は既に定義済み
- Professorは「Librarian未起動でも動作する」設計（Phase 1での後方互換）

**実装状態**: Phase 1で実装・統合完了（ローカル開発で動作確認）

---

### Phase 2: SSO認証 + 本番環境デプロイ

**Librarian責務**: Phase 1と同じ

**統合方法**: Phase 1と同じ（本番環境でもgRPC通信）

**実装状態**: Phase 1で実装済み、Phase 2で本番環境デプロイ

---

### Phase 3: Chrome Web Store公開

**Librarian責務**: Phase 1と同じ

**統合方法**: Phase 1と同じ

**実装状態**: Phase 1で実装済み、Phase 3では変更なし

---

### Phase 4: 閲覧中画面の解説機能追加

**Librarian責務**: Phase 1と同じ

**追加考慮点**:
- 画面HTML・画像解析は Professor側で実施（Gemini Vision API）
- Librarianは従来通りテキストベースの検索クエリ生成のみ

**実装状態**: Phase 1で実装済み、Phase 4では変更なし

---

### Phase 5: 学習計画立案機能（構想段階）

**Librarian責務**（未確定）:
- 学習計画生成のための推論ループ実装（可能性あり）
- 小テスト結果分析のための推論ループ実装（可能性あり）

**実装状態**: 構想段階、Phase 1-4完了後に詳細を検討

---

## Phase別の統合準備状況

| Phase | Librarian実装 | Professor統合 | 本番環境 | 備考 |
|-------|-------------|-------------|---------|------|
| **Phase 1** | ✅ 完了 | ✅ gRPC統合完了 | ❌ ローカルのみ | 推論ループ実装・検証完了 |
| **Phase 2** | ✅ 完了 | ✅ 完了 | ✅ デプロイ | Phase 1実装をそのまま本番適用 |
| **Phase 3** | ✅ 完了 | ✅ 完了 | ✅ デプロイ | Phase 1実装をそのまま維持 |
| **Phase 4** | ✅ 完了 | ✅ 完了 | ✅ デプロイ | Phase 1実装をそのまま維持 |
| **Phase 5** | ❌ 構想段階 | ❌ 構想段階 | ❌ 未定 | Phase 1-4完了後に検討 |

