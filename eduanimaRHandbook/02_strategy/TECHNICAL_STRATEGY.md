Title: Technical Strategy
Description: eduanimaR の技術方針（セキュリティ前提、検索/OCR、データ基盤、コンポーネント責務）を実装詳細抜きで定義
Owner: @OWNER
Reviewers: @reviewer1
Status: Draft
Last-updated: 2026-02-14
Tags: strategy, architecture, security, search

# Technical Strategy（実装詳細は別紙）

関連ドキュメント:
- `SERVICE_SPEC_EDUANIMA_LIBRARIAN.md`（Python: eduanima-librarian）
- `SERVICE_SPEC_EDUANIMA_PROFESSOR.md`（Go: eduanima-professor）

## 目的
本書は、eduanimaR を「学習支援（探索/要点/計画）」として成立させるための技術方針を、実装手順や細部設計抜きで定義する。

本書で決めること:
- セキュリティ/アクセス制御の前提
- 検索（全文/ベクトル/ハイブリッド）と事前処理（OCR/抽出）の基本方針
- データ基盤（GCP上のPostgreSQL + pgvector必須）の位置づけ
- コンポーネント分割（Go / Python / フロント）と責務境界

本書で決めないこと:
- API仕様、DBスキーマ、テーブル定義、具体ライブラリ、インフラ手順、コスト見積

## 大前提（非機能）
### セキュリティ
- 認証: SSO（OAuth 2.0 / OpenID Connect）を前提とする（Google / Meta / Microsoft / LINE を入口として想定）
- 認可: ユーザーごとのアクセス制限を厳格に行い、資料・生成物・ログの可視範囲を分離する
- 監査: 重要操作（取込、検索、閲覧、生成、エクスポート相当）は後追い可能にする

補足（戦略レベルの解釈）:
- 「厳格」の担保は、アプリ層だけに依存しない（DB/ストレージ/サービス境界でも分離する）
- 最小権限・ゼロトラスト志向を採用する（ネットワーク内だから安全、を前提にしない）

### プロダクト形態
- Chrome拡張機能: LMS上での導線
- Webアプリ: 検索・整理・設定・監査/履歴などの導線

補足（将来の共有）:
- 将来的に「科目の資料セット」を共有する可能性がある。
- ただし、質問履歴や重要箇所マーク等の個人の学習履歴は共有しない（プライバシー観点）。

## データ基盤
### 必須要件
- データベースは Firestore ではなく、GCP の PostgreSQL を採用する
- AlloyDB AIは使用しない（生成/ベクトル機能をマネージドで使うのではなく、PostgreSQL + pgvectorを前提に設計する）
- 抽出結果（Vision Reasoningで資料から抜き出した構造化/テキスト化結果）を、エージェントの資料検索に使用するために保存する

### ストレージの考え方
- 原本（PDF/画像等）はオブジェクトストレージへ
- 検索用の派生データ（テキスト、メタデータ、参照関係、埋め込み等）は PostgreSQL に集約する

※どの情報をどちらに置くかの厳密な定義は実装設計ドキュメントに委譲する。

## 事前処理（OCR / 抽出 / 正規化）方針
目的: Agentic Search が「grep的な全文探索」を成立させるため、資料を検索可能なテキストへ揃える。

- 画像/スキャンPDF: OCR → テキスト化
- PDF/スライド: テキスト抽出（可能なら構造保持）
- 可能なら統一フォーマット（例: Markdown相当）へ正規化
- 重要なのは“検索できる状態”であり、完全な再現やレイアウト保持は戦略上の必須ではない

## 検索戦略（全文検索 vs ベクトル検索）
### 結論（方針）
- ベースは全文検索（キーワード/固有名詞/講義用語に強い）
- セマンティック検索が必要な局面にのみベクトル検索（pgvector）を併用
- 将来的に「ハイブリッド（全文 + ベクトル）」を基本形にできる余地を残す

### なぜ「全文検索を基盤」にするか
- 配布資料は固有名詞・数式・専門用語が多く、キーワード一致が強い
- 事前OCRでテキストが整うほど、全文検索の費用対効果が上がる
- 科目ID等で絞り込みできる前提では、データ量が半年程度でも実用性能を得やすい

### なぜ「ベクトル検索も残す」か
- 同義語・言い換え・問いの抽象度が高いケースで、全文検索だけだと取り漏れやすい
- LLMが内部にベクトルモデルを持っていても、検索対象コーパス側に埋め込み（または同等の索引）がなければ、外部資料の高精度探索は困難

### pgvector（PostgreSQL拡張）の扱い
- pgvector はPostgreSQL内でベクトル型と近傍探索を提供できる
- 近似索引（HNSW/IVFFlat）を使うか、まずはフィルタ（科目ID等）+ 厳密検索で開始するかは、データ量とSLO次第で選ぶ
- 実装設計では、全文検索（tsvector等）との併用、結果融合（RRF等）や再ランキングの余地を残す

## システム構成の責務分離（Go / Python）
### Go（受付・イベント・前処理の中核）
- 認証/認可の境界（トークン検証、権限チェック、リクエストのスコーピング）
- イベント管理・キューイング（Kafka等）
- 前処理パイプライン（取込、OCR/抽出、Markdown化、Embedding生成、DB反映）

### Python（LangGraphエージェントの中核）
- LangGraphを用いたAI Agent（ツール呼び出し、探索計画、参照提示、学習ロードマップ生成など）
- 検索ツール（全文/ベクトル）の“利用側”としてのオーケストレーション

戦略的意図:
- Goは入出力/前処理/運用の堅牢性を担い、Pythonはエージェントの表現力（LangGraph）に集中する

## MCP（Model Context Protocol）を参考にすべきか
- 「エージェントが外部ツール（検索、DB、ストレージ）を安全に使うためのインターフェース設計」という観点では参考になる
- ただし、採用判断は“互換性の必要性”と“運用コスト”次第

暫定方針:
- まずはMCP的な思想（ツール境界、入力/出力スキーマ、権限制御、監査）を取り入れ、プロトコル自体の採用は後で判断する

## まとめ（02_strategyとしての決め）
- DBはGCP上PostgreSQL + pgvector（AlloyDB AIは使わない）
- 事前OCR/抽出でテキスト化し、全文検索を基盤に据える
- 必要に応じてpgvectorでセマンティック検索を併用し、将来のハイブリッドに備える
- SSO OAuth/OIDC + 厳格なユーザー別アクセス制御 + 監査を前提条件とする
- Go: 受付/キュー/前処理、Python: LangGraphエージェント

## フロントエンド技術選定（確定）
フロントエンドは、WebアプリとChrome拡張機能を同時に成立させる必要がある。
そのため「型安全性（Go⇔TS）」「コード共有（Monorepo）」「拡張機能固有制約（MV3/CSP/Service Worker）」を前提に、技術スタックを先に確定する。

### Webアプリ（確定）
- Framework: Next.js（App Router）
- Language: TypeScript
- UI System: MUI v6 + Pigment CSS
- Architecture: FSD（Feature-Sliced Design）
- Server State: TanStack Query
- API Client Gen: Orval（またはOpenAPI Generator）
- Validation: Zod
- Forms: React Hook Form
- Testing: Vitest + Playwright
- Lint/Rules: ESLint + `eslint-plugin-boundaries`（FSD境界強制）

### Chrome拡張機能（確定）
Web版の資産（FSD、MUI、型生成）を最大限共有しつつ、拡張機能としてのデファクト（ビルド/manifest/sidepanel）を優先する。

- Framework: Plasmo Framework（Manifest V3を前提）
- Language: TypeScript
- UI System: MUI v6 + Pigment CSS
	- LMS等の既存CSSとの衝突回避のため、LMS上に注入するUIはShadow DOM等の隔離戦略を前提とする
- Architecture: FSD（Webと `shared` / `entities` / `features` の分割思想を揃える）
- Server State: TanStack Query（Sidepanel/Popup等のUIでサーバー状態を統一的に扱う）
- API Client Gen: Orval（Go側のOpenAPIからTSクライアント/型を生成し、Web/拡張で共通利用）
- Communication（拡張内）: Plasmo Messaging（UI ⇔ Service Worker/Background の型安全な通信）
- DOM検知（資料の自動検知）: MutationObserver を前提に、ページ変化をトリガとして処理する

拡張機能固有の前提（実装詳細ではなく制約の確認）:
- Service Worker（Background）は常駐しない前提で設計する（起動/停止の揺らぎを許容）
- Content ScriptsはCSP/権限/注入制約があるため、機密情報の扱いとログ設計を厳格にする
- 認証はSSO（OAuth/OIDC）を前提とし、拡張からの呼び出しでもサーバー側で認可を強制する（クライアントは信用しない）

### Monorepo（前提）
- `apps/web`（Next.js）と `apps/extension`（Plasmo）を同一リポジトリで管理し、生成クライアントやUI資産を `packages/*` として共有する。

注:
- 本節は技術スタックの「確定」を目的とし、セットアップ手順や詳細なフォルダ構成は実装詳細ドキュメントへ委譲する。
