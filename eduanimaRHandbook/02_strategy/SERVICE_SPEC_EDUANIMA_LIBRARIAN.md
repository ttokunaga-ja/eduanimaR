Title: Service Spec — eduanima-librarian (Python)
Description: Pythonマイクロサービス「The Librarian」の責務・技術選定・インターフェース仕様（実装詳細除く）
Owner: @OWNER
Reviewers: @reviewer1
Status: Draft
Last-updated: 2026-02-14
Tags: strategy, service-spec, agent, python

# eduanima+R: Python Microservice Specification (The Librarian)

## 0. サマリ
`eduanima-librarian`（Librarian）は、システム全体の「探索・根拠収集」の中核です。
最終回答の生成は責務に含めず、膨大な講義資料から学習に資する根拠（エビデンス）を自律的に探索し、最小で強い資料セットを構成して `eduanima-professor` に返します。

## 1. 役割：専門司書（The Librarian）
Librarianのゴール:
- ユーザーの質問に対して、回答生成に必要な根拠を「見つける・揃える・過不足を評価する」
- 根拠の所在（資料ID/ページ/断片）を構造化して返す

Librarianの非ゴール:
- 最終的な学習支援文の生成（解説・ロードマップ本文の生成）は行わない
- DBへ直接アクセスしない（検索も取得もGo経由）
- 永続状態を持たない（原則ステートレス）

## 2. 核心的責務
### 2.1 自律的検索戦略の立案（Agentic Search Strategy）
- ユーザーの曖昧な質問を解析し、PostgreSQLの「全文検索」と「ベクトル検索（pgvector）」に適した複数の検索要求を組み立てる
- 1回の検索に固定せず、探索→評価→再探索のループを前提にする

### 2.2 検索結果の評価とリフレクション（Search Evaluation & Reflection）
- Goから返却された検索結果（Markdown断片・メタデータ）を評価し、質問への充足度を判定する
- 不足要素（前提知識、定義、数式、具体例、条件分岐など）を特定し、次の検索要求に反映する

### 2.3 資料の選別と重複排除（Deduplication & Refinement）
- ループで集まった候補から、関連性が高い資料ID/ページを特定する
- 断片（chunk）の重複を排除し、回答生成に必要な「最小かつ最強の資料セット」を構成する

### 2.4 DBレス・ステートレス推論（Stateless Inference）
- DB接続を持たず、必要な状態（会話履歴、検索履歴、既取得断片、現在の探索ステップ等）はリクエストごとに受け取る
- 返却するのは「次に取るべき探索アクション」または「収集完了した根拠セット」

## 3. 技術スタックと選定理由
### 3.1 Webフレームワーク: Litestar
- 選定理由: DBレスの推論サービスにおいて、型安全なDTOと高速なシリアライズで通信オーバーヘッドを抑えるため
- 期待: コントローラ/DTOでインターフェース契約を強制し、GoとのI/F不整合を減らす

### 3.2 エージェント制御: LangGraph
- 選定理由: 「検索→評価→再検索」の反復をステートマシンとして安全に管理するため
- 要件: 最大ループ数、条件分岐、探索状態（state）の明示

### 3.3 推論モデル: Gemini（Tool Use対応の高速モデル）
- 用途: 検索要求の生成、検索結果評価、欠落情報の特定
- 選定理由: Tool Use（Function Calling）で構造化出力を安定させ、探索の収束性を高めるため

### 3.4 データ検証/高速化: msgspec
- 選定理由: 大きな検索結果のJSON入出力を高速化し、レイテンシを削減するため

## 4. インターフェース設計（Go <-> Python）
### 4.1 公開エンドポイント
- `POST /agent/librarian/think`

このエンドポイントは「思考の1ステップ」を実行し、次のアクションを返します。

### 4.2 Input（Go -> Python）
最小フィールド（案）:
- `request_id`: トレーシング用ID
- `user_query`: ユーザーの質問（テキスト）
- `subject_id`: 科目ID（Go側で強制される絞り込み条件。文脈理解にも利用）
- `search_history`: これまでの検索要求と結果・評価の履歴
- `state`: 探索ステップ状態（LangGraph stateをシリアライズしたもの）
- `constraints`: ループ上限、返却件数上限、許容レイテンシなど

注: ユーザーID等の認可情報はLibrarianが意思決定に必要な範囲で参照してもよいが、認可の強制はGoが担う（Librarianは信用しない）。

### 4.3 Output（Python -> Go）
Librarianは以下のいずれかを返します。

#### (A) 検索要求（Action=`SEARCH`）
意図: 「この条件で検索して結果を返してほしい」
- `action`: `SEARCH`
- `queries_text`: 全文検索用クエリ群（複数）
- `queries_vector`: ベクトル検索用クエリ群（複数・任意）
- `filters_hint`: 絞り込みの意図（subject_id前提の追加ヒント。強制はGo側）
- `rationale`: なぜこの検索を行うか（監査・デバッグ用の短い説明）

#### (B) 収集完了（Action=`COMPLETE`）
意図: 「根拠は揃った。次工程（合成）へ渡せ」
- `action`: `COMPLETE`
- `evidence`: 以下を含む構造化配列
  - `document_id`
  - `pages`（任意）
  - `snippets`（Markdown断片）
  - `score`（任意）
  - `why_relevant`（短文）
- `coverage_notes`: 充足している点/不確実な点のメモ（短文）

#### (C) 失敗（Action=`ERROR`）
意図: 「これ以上探索できない」
- `action`: `ERROR`
- `error_type`: `LOOP_LIMIT` | `TIMEOUT` | `INVALID_INPUT` | `MODEL_FAILURE` | `OTHER`
- `message`

## 5. ガードレール（必須）
- 最大ループ数: LangGraphで上限（例: 3）を強制し、無限ループを防止する
- DBアクセス禁止: PostgreSQLドライバ等を依存に含めない（検索・取得は全てGo経由）
- 出力制約: 返却は常に「検索要求」または「根拠セット」の構造化データに限定する
- 監査性: `rationale` や評価メモを返し、なぜその根拠を選んだか追跡可能にする

## 6. 非機能要件（簡易）
- ステートレス: スケールアウト前提（Cloud Run等）
- タイムアウト: 1ステップの推論時間上限を設ける（詳細値は別紙）
- 観測性: `request_id`でログ/トレースを相関可能にする

## 7. Open Questions（要確認）
- 検索結果（Go→Python）の返却形式: snippet粒度、最大件数、メタデータ項目
- `queries_vector` の扱い: Librarianが埋め込みを生成するか、Go側で生成するか（現方針ではGo側前処理が主）
- 「ページ番号」の扱い: PDF由来のページとMarkdown断片の対応付けの定義
