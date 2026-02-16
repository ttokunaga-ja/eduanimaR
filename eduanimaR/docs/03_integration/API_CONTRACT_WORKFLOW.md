---
Title: API Contract Workflow
Description: eduanimaRのAPI契約管理とOpenAPI運用フロー
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-15
Tags: frontend, eduanimaR, api, openapi, orval, professor
---

# API Contract Workflow（OpenAPI / Orval）

Last-updated: 2026-02-15

このドキュメントは、バックエンド（Go Gateway / Microservices）との **API契約（OpenAPI）** を、
フロントエンド（Next.js + FSD）で **安全に運用するための手順と禁止事項** を固定します。

関連：
- 生成手順：`API_GEN.md`
- データ取得の契約：`../01_architecture/DATA_ACCESS_LAYER.md`

---

## 結論（Must）

- OpenAPI は **契約のSSOT**（フロントで推測実装しない）
- 固定値（enum）は **API契約（OpenAPI）で定義した enum を SSOT とし、生成物（Orval等）でフロントに取り込む**
  - バックエンドでは可能な箇所で **DB の ENUM（PostgreSQL ENUM）を採用**し、API / DB / アプリケーション間で enum の意味とマッピングを明確に保つ
  - フロントで TypeScript の `enum` を使用するかどうかは、変更頻度・互換性（未知値の扱い）・運用性を考慮して判断する
  - **独自に固定値を定義することは原則禁止**
- クライアントは **Orval生成物のみ** を入口にする（手書き `fetch/axios` 禁止）
- 破壊的変更は **バージョニング/廃止手順** に従う（`API_VERSIONING_DEPRECATION.md`）
- 生成物は **CIで差分検知** し、契約ズレを早期に止める

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

SSOT の優先順位：
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
