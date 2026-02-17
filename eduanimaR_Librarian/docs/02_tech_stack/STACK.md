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

## Phase別のLibrarian実装スケジュール

### Phase 1: Librarian未実装

**Phase 1では本サービス（Librarian）は未実装です。**

- Phase 1（ローカル開発・認証スキップ）では、Professorが直接Gemini 2.0 Flashを呼び出す
- Librarian推論ループは実装しない
- Professor単体で以下を実現:
  - OCR + 構造化処理（Gemini 2.0 Flash）
  - 単純な検索（pgvector HNSW）
  - 回答生成（Gemini 2.0 Flash）

### Phase 2: Librarian未実装（継続）

**Phase 2でもLibrarianは不要です。**

- SSO認証実装
- 拡張機能版ZIPファイル配布
- 本番環境デプロイ（Professor単体）
- Librarian推論ループは実装しない

### Phase 3: Librarian実装・統合

**Phase 3で初めてLibrarianを実装・統合します。**

#### Phase 3での責務

- **検索戦略立案（Plan）**: Professorから受け取った指示をもとに検索方針を立案
- **検索結果の評価・停止判断（Evaluate/Decide）**: 内容評価と終了条件の判定
- **エビデンス選定（Rank）**: 根拠資料の抽出・評価

#### Phase 3での統合準備

1. **gRPC契約の実装**
   - 契約SSOT: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
   - Professor側: gRPCクライアント実装
   - Librarian側: gRPCサーバー実装

2. **Professorの後方互換性**
   - Librarian未起動でも動作する設計（Phase 1/2との互換性）
   - Librarian接続失敗時はProfessor単体で検索を実行

3. **LangGraph導入**
   - 検索ループの状態管理
   - MaxRetry/停止条件の保証
   - Gemini 3 Flash での推論実行

#### Phase 3での検証項目

- Librarian推論ループの動作確認（最大5回試行）
- Professor ↔ Librarian gRPC双方向ストリーミング検証
- 検索精度の向上確認（Phase 1/2 vs Phase 3）

### Gemini 3 Flash の使い分け（Phase 3以降）

- **戦略立案（Plan）**: 思考コストを許容（例: medium 相当）
- **ループ中の微修正（Refine/Evaluate）**: 低コスト（例: low 相当）

**注**: 具体のパラメータ名/SDKは採用SDKに依存するため、実装側でSSOT化する。

---

### Phase 4: 閲覧中画面の解説機能追加

**Librarian責務**: Phase 3と同じ

**追加考慮点**:
- 画面HTML・画像解析は Professor側で実施（Gemini Vision API）
- Librarianは従来通りテキストベースの検索クエリ生成のみ

**実装状態**: Phase 3で実装済み、Phase 4では変更なし

---

### Phase 5: 学習計画立案機能（構想段階）

**Librarian責務**（未確定）:
- 学習計画生成のための推論ループ実装（可能性あり）
- 小テスト結果分析のための推論ループ実装（可能性あり）

**実装状態**: 構想段階、Phase 1-4完了後に詳細を検討

