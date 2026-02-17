---
Title: API Contract Workflow
Description: eduanimaRのAPI契約管理とOpenAPI運用フロー
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, api, openapi, orval, professor
---

# API Contract Workflow（OpenAPI / Orval）

Last-updated: 2026-02-16

このドキュメントは、バックエンド（Go Gateway / Microservices）との **API契約（OpenAPI）** を、
フロントエンド（Next.js + FSD）で **安全に運用するための手順と禁止事項** を固定します。

関連：
- 生成手順：`API_GEN.md`
- データ取得の契約：`../01_architecture/DATA_ACCESS_LAYER.md`

---

## 結論（Must）

### 3層契約の整理

**1. OpenAPI（外部契約: Frontend ↔ Professor）**
- 場所: `eduanimaR_Professor/docs/openapi.yaml`
- 責務: HTTPエンドポイント、リクエスト/レスポンス型、enum定義
- フロントエンド: Orvalで生成されたクライアント/型のみ使用（手書きfetch/axios禁止）

**2. gRPC/Proto（内部契約: Professor ↔ Librarian）**
- 場所: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
- 責務: 探索開始/評価ループ/検索要求/探索完了通知
- フロントエンド: 直接関与しない（Professorが仲介）

**3. DB ENUM（データ層契約: PostgreSQL）**
- 場所: PostgreSQL ENUM型
- 責務: API/DB/アプリケーション間でenumの意味を一致（SSOT）
- フロントエンド: OpenAPI経由で取得した値のみ使用（独自定義禁止）

### enum運用方針（SSOT）

**バックエンド（Professor）の責務**:
1. PostgreSQL ENUM型で定義（例: `CREATE TYPE search_strategy AS ENUM ('keyword', 'semantic', 'hybrid')`）
2. OpenAPI定義で公開（`components.schemas` に enum として記載）
3. API/DB/アプリケーション間で意味を一致させる

**フロントエンドの責務**:
1. Orvalで生成された型のみ使用（`type SearchStrategy = 'keyword' | 'semantic' | 'hybrid'`）
2. **未知値への対応**: switch文で必ずdefault句を用意（exhaustive checkは禁止）
3. **独自enum定義の禁止**: バックエンドで未定義のenum値をフロントで追加しない

**悪い例**（❌ 禁止）:
```typescript
// フロントで独自にenum値を追加
type SearchStrategy = 'keyword' | 'semantic' | 'hybrid' | 'custom'; // ❌ 'custom'はDB/APIに存在しない

// exhaustive check（未知値でエラー）
switch (strategy) {
  case 'keyword': return ...;
  case 'semantic': return ...;
  case 'hybrid': return ...;
  // default句なし → 将来のenum追加で破壊
}
```

**良い例**（✅ 推奨）:
```typescript
// Orval生成型のみ使用
type SearchStrategy = 'keyword' | 'semantic' | 'hybrid'; // OpenAPIから生成

// default句で未知値を許容
switch (strategy) {
  case 'keyword': return ...;
  case 'semantic': return ...;
  case 'hybrid': return ...;
  default:
    console.warn(`Unknown strategy: ${strategy}`);
    return fallbackBehavior(); // フォールバック動作
}
```

---

## Professor/Librarianとの通信パターン（Librarian推論ループ）

### Librarian推論ループの概要

Librarian は LangGraph による自律的な推論ループを実行し、最大5回の反復で検索戦略を立案・修正します：

**フェーズ**:
1. **Plan/Refine**: 検索戦略立案、クエリ生成（初回はPlan、2回目以降はRefine）
2. **Search Tool**: Professor経由で検索実行（HTTP/JSON）
3. **Evaluate**: 検索結果から選定エビデンス抽出、充足度評価
4. **Route**: 停止条件判定（COMPLETE → 終了、CONTINUE → Planに戻る、ERROR → エラー）

**停止条件**:
- 十分なエビデンスが収集された（COMPLETE）
- 最大試行回数（5回）に到達（MAX_RETRIES_REACHED）
- エラー発生（ERROR）

### フェーズごとのモデル選定（環境変数）

| フェーズ | 責務 | モデル | 環境変数 |
|:---|:---|:---|:---|
| Phase 1 | Ingestion（構造化） | Gemini 3 Flash（Batch） | `PROFESSOR_GEMINI_MODEL_INGESTION` |
| Phase 2 | Planning（大戦略） | Gemini 3 Flash | `PROFESSOR_GEMINI_MODEL_PLANNING` |
| Phase 3 | Search（小戦略） | Gemini 3 Flash | `LIBRARIAN_GEMINI_MODEL_SEARCH` |
| Phase 4 | Answer（最終回答） | Gemini 3 Pro | `PROFESSOR_GEMINI_MODEL_ANSWER` |

### Phase 3の停止条件（Definition of Done）

Librarianが探索を終了する条件（Phase 2で定義、Phase 3で判定）:

1. **充足性（Sufficiency）**: 必須項目（式/定義/ケース等）が根拠と紐付いて揃っている
2. **明確性（Unambiguity）**: 近似概念（相関係数など）と混同していない
3. **視覚情報の言語化（Visual Check）**: 図表の凡例/線種/注記など"指しているもの"が確保できている

または：
- MaxRetry（5回）到達
- タイムアウト（60秒）

**参照**: [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md)、[`../../eduanimaR_Librarian/docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`](../../eduanimaR_Librarian/docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md)

### SSEストリーミング要件（Librarian推論ループの進捗イベント）

Professor SSEは、Librarian推論ループの進捗を以下のイベントでリアルタイム配信します：

| イベントタイプ | 内容 | UI反映 |
|:---|:---|:---|
| `thinking` | Phase 2実行中（タスク分割・停止条件生成） | 「AI Agentが検索方針を決定しています」 |
| `searching` | Librarian推論ループ実行中 | プログレスバー（例：「2/5回目の検索」） |
| `evidence` | 選定エビデンス提示 | エビデンスカード表示 |
| `answer` | 最終回答生成中 | リアルタイムテキスト追加 |
| `done` | 完了 | SSE接続を閉じる |
| `error` | エラー | エラートースト |

**search_loop_progress イベントの詳細**:
```json
{
  "type": "search_loop_progress",
  "request_id": "req_abc123",
  "node": "Plan",
  "status": "SEARCHING",
  "current_retry": 2,
  "max_retries": 5,
  "message": "検索クエリ生成中...",
  "timestamp": "2026-02-16T12:34:58Z"
}
```

**参照**: [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md)

### エラーハンドリング（監査ログ・request_id伝播）

**監査ログ要件**:
- すべてのAPIリクエストに`request_id`を付与（`X-Request-ID`ヘッダー）
- Professor → Librarian推論ループでも`request_id`を伝播
- エラーレスポンスに`request_id`を含める
- ログ横断検索で原因調査が可能

**request_id伝播フロー**:
1. Frontend → Professor: `X-Request-ID`ヘッダー
2. Professor → Librarian: gRPC metadata で `request_id` 伝播
3. Professor → Frontend: SSEイベント・エラーレスポンスに`request_id`を含める

**エラーレスポンス例**:
```json
{
  "error": {
    "code": "LIBRARIAN_TIMEOUT",
    "message": "Librarian推論ループがタイムアウトしました",
    "request_id": "req_abc123"
  }
}
```

**参照**: [`../../eduanimaRHandbook/01_philosophy/PRIVACY_POLICY.md`](../../eduanimaRHandbook/01_philosophy/PRIVACY_POLICY.md)、[`../../eduanimaR_Professor/docs/05_operations/OBSERVABILITY.md`](../../eduanimaR_Professor/docs/05_operations/OBSERVABILITY.md)

---

## enum と UI文言の扱い（重要）

### 禁止事項（MUST NOT）

以下の実装パターンは **互換性・可読性・保守性を著しく低下させる**ため禁止：

1. **API の enum を独自に文字列や数値（Int）に変換して管理すること**
2. **プロジェクト内で散逸するマッピングを持つ実装**
3. **ハードコーディングされた UI 文言を直接画面に書くこと**

### 正しい運用（MUST）

#### enum の扱い
- 必要な変換やマッピングは **サーバ側で行う** か、**明示的でテストされたアプリケーション層のマッピングレイヤ**で実装する
- マッピングは中央管理され、散逸しないこと

#### UI 文言の扱い
- **画面表示（UI 文言）は必ず i18n を使用**し、表示する全ての文字列を翻訳キー（変数）化して、各言語ごとの JSON ファイルから読み出す
- API 由来の enum/ラベルを表示する場合は、**表示用の翻訳キーに明示的にマッピング**し、そのマッピングは中央管理かつテスト済みであること
- 翻訳ファイルの不足や未使用キーは **CI（ビルド/テスト）で検出する仕組み**を整えること

詳細は `I18N_LOCALE.md` を参照。

---

## 1) SSOT（Single Source of Truth）

**契約の種類と管理**:
- **外部API（Frontend ↔ Professor）**: OpenAPI（`openapi/openapi.yaml`）
- **内部RPC（Professor ↔ Librarian）**: gRPC/Proto（`eduanimaR_Professor/proto/librarian/v1/librarian.proto`）

**フロントエンドのSSOT優先順位**：
1. OpenAPI（バックエンド）
2. Orval生成物（フロントの型/クライアント）
3. フロント実装（UI/画面）

禁止：
- OpenAPI から読み取れない仕様をフロント側で“推測”して実装する
- 生成物を手編集する（次回生成で消える）

---

## 2) 契約変更の基本フロー

### 2.1 追加（後方互換）
- 例：レスポンスに optional フィールド追加、enum に値追加（互換注意）
- 手順：
  1. BE: OpenAPI更新
  2. FE: `npm run api:generate` で生成物更新
  3. FE: UI は **tolerant reader**（未存在/追加を許容）
  4. 必要なら Feature Flag で段階的に露出

### 2.2 変更（互換性あり/なしを明確化）
- 変更の種類を分類し、互換が無い場合は **廃止手順** に寄せる

### 2.3 削除（破壊的）
- 原則禁止（どうしても必要なら `API_VERSIONING_DEPRECATION.md` に従う）

---

## 3) operationId と生成の安定性

生成される hook/client 名を安定化するために、OpenAPI の `operationId` は以下を推奨：
- 一度決めたら変更しない（命名変更は破壊的変更になりうる）
- `VerbNoun` で統一（例：`GetUser`, `UpdateUserEmail`）

---

## 4) 契約レビュー観点（チェックリスト）

- OpenAPI の変更は **ユースケース視点で妥当**か（画面要件と整合）
- 追加/変更/削除が **後方互換** か（互換でないなら廃止手順）
- エラー形式・エラーコードが契約化されているか（`ERROR_HANDLING.md` / `ERROR_CODES.md`）
- 認証/認可の前提（Cookie/Bearer、必要スコープ）が明文化されているか

---

## 5) CI で止める（推奨）

目的：ローカルで生成し忘れて「型ズレを見逃す」を防ぐ。

- CIで `npm run api:generate` を実行し、生成物に差分が出ないことを確認する
- 差分が出たら、OpenAPI または生成設定を正して再生成する

---

## 6) 禁止（AI/人間共通）

- 画面内での手書き `fetch/axios`（生成物がある前提）
- “OpenAPIに無い” エンドポイント/フィールドの利用
- `generated/` の手編集

---

## eduanimaR 固有の契約運用

### OpenAPI の正（SSOT）
- **定義元**: Professor（Go）の `docs/openapi.yaml`
- **配置**: 本リポジトリの `openapi/openapi.yaml` にコピー（CI で差分検出）
- **生成**: Orval で `src/shared/api/generated/` に TypeScript コード生成

### SSE（Server-Sent Events）契約
Professor の `/qa/stream` エンドポイントは以下のイベント型を配信：

| イベント型 | 内容 | 発信元 |
|:---|:---|:---|
| `plan` | 調査項目・停止条件 | Professor（Phase 2: Plan生成） |
| `search` | 検索結果（クエリ、ヒット件数） | Librarian Agent（検索戦略実行） |
| `answer` | 最終回答（本文 + ソース） | Professor（Phase 2: Gemini 3 Flash） |
| `error` | エラー通知（`ERROR_CODES.md` の code を含む） | Professor / Librarian |
| `done` | 完了通知 | Professor |

**クライアント側の実装要件**:
- 接続断・再接続を前提にする（`EventSource` の `error` イベントをハンドリング）
- イベントの重複を許容する設計（idempotency）
- `error` イベント受信時は `ERROR_CODES.md` に基づいて UI を更新

### エラーコードの同期
- Professor の `ERROR_CODES.md` を SSOT とし、フロントエンドの `03_integration/ERROR_CODES.md` に同期
- エラー UI は `03_integration/ERROR_HANDLING.md` の方針に従う
- **同期頻度**: Professor のエラーコード追加・変更時、即座にフロントエンド側を更新（CI で差分検出）

### Breaking Changes の扱い
- Professor が以下の変更を行う場合、事前に Slack/Issue で通知：
  - 必須フィールド追加
  - 型変更（string → number 等）
  - エンドポイント削除
  - SSE イベント型の変更
- フロントエンドはマイグレーション期間（1週間）を設ける
- 期間中は旧・新両方のスキーマをサポート（後方互換）

### バージョニング
- OpenAPI の `version` フィールドを SSOT とする
- メジャーバージョンアップ（v1 → v2）時は、フロントエンドの API クライアント生成を再実行
- マイナー・パッチバージョンは後方互換を保証

---

## Professor OpenAPIがSSOT

### 契約の場所
- **SSOT**: `eduanimaR_Professor/docs/openapi.yaml`
- **管理**: Professor（Go）リポジトリ
- **フロントエンド**: 自動生成クライアント（Orval）

### OpenAPI更新フロー

1. **Professor側で更新**
   - `docs/openapi.yaml` を修正
   - Breaking Changesを明記（コメント or CHANGELOG）

2. **フロントエンド側で対応**
   - OpenAPI取得: `curl https://professor.example.com/openapi.yaml > openapi.yaml`
   - Orval再生成: `npm run api:generate`
   - 差分確認: `git diff src/shared/api/generated/`
   - 必要に応じてコード修正

3. **CI/CDで整合性チェック**
   - Orval再生成を実行
   - 差分があればCIエラー

### Breaking Changes対応

#### 例: 必須フィールド追加
```yaml
# Before
QuestionRequest:
  type: object
  properties:
    text: string

# After
QuestionRequest:
  type: object
  required:
    - text
    - subjectId  # 新規必須
  properties:
    text: string
    subjectId: string
```

#### 移行計画
1. Professor側で`subjectId`を任意フィールドとして追加（v1.1）
2. フロントエンド側で対応（3ヶ月猶予）
3. Professor側で必須化（v2.0）

### APIバージョニング
- **形式**: `/v1/`, `/v2/`
- **移行期間**: 旧バージョンは6ヶ月サポート
- **廃止通知**: レスポンスヘッダー `X-API-Deprecated: true`

---

## Professor OpenAPI更新フロー

### 更新フロー
1. **Professor側でOpenAPI更新**
   - Professor側で`openapi.yaml`更新
   - Breaking Changesを明記（CHANGELOG or コメント）
   - Git tagでバージョン管理

2. **フロントエンド側で生成物更新**
   - `npm run api:generate`実行
   - 生成物の差分確認: `git diff src/shared/api/generated/`
   - Breaking Changesの場合はフロントエンド修正

3. **更新頻度**
   - Phase 1: 週次（機能開発が活発なため）
   - Phase 2以降: 月次（安定運用のため）
   - 緊急修正（バグ修正、セキュリティパッチ）: 随時

4. **CI/CDで整合性確認**
   - Orval再生成を実行
   - 差分があればCIエラー
   - 差分がある場合の対応: OpenAPIまたは生成設定を修正して再生成

### Breaking Changes対応例
Breaking Changesには以下が含まれます：
- 必須フィールド追加
- 型変更（string → number 等）
- エンドポイント削除
- SSE イベント型の変更

**移行計画**:
1. Professor側で任意フィールドとして追加（v1.1）
2. フロントエンド側で対応（3ヶ月猶予）
3. Professor側で必須化（v2.0）

---

## Librarian ↔ Professor の契約（HTTP/JSON）

### エンドポイント: `POST /v1/search-tool`
**責務**: Librarianの検索依頼を受け、ハイブリッド検索（RRF）結果を返す。

#### リクエスト（Librarian → Professor）
```json
{
  "request_id": "req_abc123",
  "keyword_list": ["決定係数", "定義"],
  "semantic_query": "決定係数の計算式について",
  "exclude_ids": ["chunk_001", "chunk_005"],
  "metadata_filters": {
    "subject_id": "subj_xyz",
    "user_id": "user_123"
  }
}
```

**フィールド説明**:
- `keyword_list`: 全文検索（BM25）用キーワード配列
- `semantic_query`: ベクトル検索用の自然言語クエリ
- `exclude_ids`: 既読チャンクID（DB層で`NOT IN`除外）
- `metadata_filters`: Professor側で認可チェック後に強制適用

#### レスポンス（Professor → Librarian）
```json
{
  "chunks": [
    {
      "temp_index": 0,
      "text": "## 決定係数\n定義: 回帰モデルの説明力を示す指標...",
      "metadata": {
        "page": 3,
        "heading": "回帰分析の評価指標"
      }
    }
  ],
  "total_searched": 50,
  "current_retry": 2
}
```

**フィールド説明**:
- `temp_index`: Professorが一時的に割り当てる番号（LLMのハルシネーション防止）
- `text`: Markdown形式のチャンク本文
- `total_searched`: ハイブリッド検索で探索した総チャンク数
- `current_retry`: Librarian推論ループの現在の試行回数

### ハイブリッド検索（RRF）の実行フロー
1. **並列検索**: `keyword_list`でBM25、`semantic_query`でpgvector検索を同時実行
2. **RRF統合**: 各検索結果の順位（Rank）から統合スコアを計算
   - 公式: `Score = 1/(60 + Rank_vector) + 1/(60 + Rank_keyword)`
   - k定数=60は業界標準値
3. **動的k値調整**: 母数N（全チャンク数）と`retry_count`に基づき取得件数を決定
   - 例: N < 1,000 → k=5, N ≥ 100,000 → k=20
4. **除外処理**: `exclude_ids`をSQL `WHERE id NOT IN (...)`で適用
5. **返却**: RRFスコア上位k件を`temp_index`付きで返却

---

## SSE（Server-Sent Events）契約

### Professor SSEエンドポイント
- **エンドポイント**: `GET /v1/search/stream?query={query}&subject_id={subject_id}`
- **認証**: Cookie（SSO/OAuth）または`Authorization: Bearer {token}`
- **Content-Type**: `text/event-stream`

### SSEイベントタイプ

| イベントタイプ | 内容 | Phase | 発信元 |
|:---|:---|:---|:---|
| `answer_chunk` | 回答断片（ストリーミング配信） | Phase 2+ | Professor（Gemini 2 Flash） |
| `evidence_selected` | 選定エビデンス（Librarianが選定した根拠箇所） | Phase 3+ | Professor（Librarian推論結果を変換） |
| `search_loop_progress` | Librarian推論ループの進行状況 | Phase 3+ | Professor（Librarianの中間状態を中継） |
| `error` | エラー通知（`ERROR_CODES.md`のcodeを含む） | All | Professor / Librarian |
| `done` | 完了通知 | All | Professor |

### イベントデータ構造

#### `answer_chunk`
```json
{
  "type": "answer_chunk",
  "request_id": "req_abc123",
  "chunk": "回答の断片テキスト",
  "timestamp": "2026-02-16T12:34:56Z"
}
```

#### `evidence_selected`
```json
{
  "type": "evidence_selected",
  "request_id": "req_abc123",
  "evidence": {
    "document_id": "doc_xyz789",
    "snippets": ["## 見出し\n本文の断片..."],
    "why_relevant": "この箇所は質問に対する直接的な回答を含むため"
  },
  "timestamp": "2026-02-16T12:34:57Z"
}
```

#### `search_loop_progress`
```json
{
  "type": "search_loop_progress",
  "request_id": "req_abc123",
  "node": "Plan",
  "status": "SEARCHING",
  "current_retry": 2,
  "max_retries": 5,
  "message": "検索クエリ生成中...",
  "timestamp": "2026-02-16T12:34:58Z"
}
```

**フィールド説明**:
| フィールド | 型 | 説明 |
|:---|:---|:---|
| `node` | string | 現在のLangGraphノード（Plan/Search/Evaluate/Route） |
| `status` | string | ノードの状態（SEARCHING/EVALUATING/COMPLETE） |
| `current_retry` | number | Librarian推論ループの現在の試行回数 |
| `max_retries` | number | 推論ループの上限回数（デフォルト: 5） |
| `message` | string | ユーザー向け進捗メッセージ（例: "検索クエリ生成中..."） |

**LangGraphノードの説明**:
- **Plan/Refine**: 検索戦略立案、クエリ生成（初回はPlan、2回目以降はRefine）
- **Search Tool**: Professorの`POST /v1/search-tool`呼び出し
- **Evaluate**: 検索結果から`evidence_snippets`抽出、充足度評価
- **Route**: 停止条件判定（`COMPLETE`なら終了、`CONTINUE`ならPlanに戻る）

**フロントエンド実装要件**:
- `node`フィールドで現在のノード名を表示（例: "検索戦略を立案中..."）
- `current_retry / max_retries`でプログレスバー更新（例: "2/5 回目の検索"）
- `status=COMPLETE`で次イベント（`evidence_selected`）を待機
- `message`をユーザーに表示（UIフィードバック）

### フロントエンド側の処理

#### EventSourceでの受信
```typescript
const eventSource = new EventSource(`/v1/search/stream?query=${query}&subject_id=${subjectId}`);

eventSource.addEventListener('answer_chunk', (event) => {
  const data = JSON.parse(event.data);
  appendAnswerChunk(data.chunk);
});

eventSource.addEventListener('evidence_selected', (event) => {
  const data = JSON.parse(event.data);
  displayEvidence(data.evidence);
});

eventSource.addEventListener('search_loop_progress', (event) => {
  const data = JSON.parse(event.data);
  updateProgressBar(data.current_retry, data.max_retries);
});

eventSource.addEventListener('error', (event) => {
  const error = JSON.parse(event.data);
  handleError(error.code);
});

eventSource.addEventListener('done', () => {
  eventSource.close();
});
```

#### TanStack Queryでの状態管理
```typescript
import { useQuery } from '@tanstack/react-query';

export function useSearchStream(query: string, subjectId: string) {
  return useQuery({
    queryKey: ['search', 'stream', subjectId, query],
    queryFn: async () => {
      // SSE接続を確立し、イベントを購読
      const stream = await connectSSE(`/v1/search/stream`, { query, subjectId });
      return stream;
    },
    staleTime: 5 * 60 * 1000, // 5分
  });
}
```

---

## Librarian結果の受信

### Professorの変換処理
1. **LibrarianからProfessorへ**: Librarianは`selected_evidence`（`temp_index`配列）を返す
2. **ProfessorでID変換**: Professorが`temp_index`を安定ID（`document_id`）に変換
3. **ProfessorからFrontendへ**: `document_id` + `snippets`を含む`evidence_selected`イベントを配信

### フロントエンドの責務
- **`temp_index`を意識しない**: フロントエンドは`temp_index`を直接扱わない
- **`document_id` + `snippets`を表示**: Professorが変換済みのIDと断片を表示
- **`entities/evidence`で管理**: 選定エビデンスは`entities/evidence`レイヤーで状態管理

### データフロー図
```
Librarian → Professor → Frontend
  ↓            ↓          ↓
temp_index  変換処理  document_id + snippets
(一時ID)              (安定ID)
```
