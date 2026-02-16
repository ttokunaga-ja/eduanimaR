# Docs Portal（Frontend / FSD Template）

Last-updated: 2026-02-16

この `docs/` 配下は、Next.js（App Router）+ FSD（Feature-Sliced Design）での開発を「契約（運用ルール）」として固定するためのドキュメント集です。

目的：
- 判断のぶれ（人間/AI）を減らす
- 依存境界・契約駆動・運用の事故を先に潰す
- "本番だけ壊れる" を再現可能な手順に落とす

## サービス全体のコンセプト

### Mission & North Star
- **Mission**: 学習者が「探す時間を減らし、理解に使う時間を増やせる」学習支援ツール
- **North Star Metric**: 資料から根拠箇所に到達するまでの時間短縮

### Professor / Librarian の役割
- **Professor（Go）**: データ所有者、最終回答生成、DB/GCS/Kafka の物理実行を担当
- **Librarian（Python）**: 検索戦略立案、LangGraph Agent による推論（Professor経由でのみ検索実行）

### 上流ドキュメントへの参照
- サービスコンセプト全体: [`../../eduanimaRHandbook/README.md`](../../eduanimaRHandbook/README.md)
- バックエンド Professor: [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)
- バックエンド Librarian: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)

---

## Quickstart（最短で開発開始）
0. `00_quickstart/QUICKSTART.md`（30分で着手できる状態にする）
1. `00_quickstart/PROJECT_DECISIONS.md`（プロジェクト固有の決定事項SSOT）

**重要な前提（Phase構成）**:
- **Phase 1（開発環境）**: 
  - ローカルでの動作確認のみ
  - 認証なし（dev-user固定）
  - 自動アップロード機能の実装と検証
  - Web版: curlやPostmanでAPIテスト
  - 拡張機能: Chromeにローカル読み込みで動作確認
  
- **Phase 2（本番環境・同時リリース）**:
  - SSO認証実装（Google/Meta/Microsoft/LINE）
  - Chrome Web Storeへ公開（非公開配布）
  - Webアプリの本番デプロイ
  - **Web版からの新規登録は禁止、拡張機能でのみユーザー登録可能**
  - **Web版で新規ユーザーのログイン試行を検知した場合、以下へ誘導**：
    1. Chrome Web Store（拡張機能公式ページ）
    2. GitHubリリースページ（代替ダウンロード）
    3. 公式導入ガイド・解説ブログ
  
- **ファイルアップロード**: 
  - フロントエンドにUIを実装してはならない
  - Phase 1: API直接呼び出し + 拡張機能実装
  - Phase 2: 拡張機能の自動アップロードのみ

## まず読む（最短ルート）
1. **プロジェクト固有の前提**: `00_quickstart/PROJECT_DECISIONS.md` ← **最優先**
2. 技術スタック（SSOT）：`02_tech_stack/STACK.md`

## 認証とユーザー登録の境界（Phase 2）

### ユーザー登録フロー
- **新規登録**: Chrome拡張機能でのSSO認証のみ許可
- **既存ユーザーのログイン**: Web版でも可能（拡張機能で登録済みのユーザーのみ）

### Web版での未登録ユーザー対応
Web版でSSO認証後、未登録ユーザーと判定された場合：
1. **登録不可の通知**を表示
2. **拡張機能ダウンロードページへ誘導**（優先順位順に表示）:
   - Chrome Web Store: `https://chrome.google.com/webstore/detail/[extension-id]`
   - GitHub Releases: `https://github.com/[org]/[repo]/releases`
   - 公式導入ガイド: `[ブログURL]` または `[公式ドキュメント]`
3. **誘導UI**:
   - タイトル: 「eduanimaRをご利用いただくには、Chrome拡張機能のインストールが必要です」
   - 説明: 「Web版は既存ユーザーのログイン専用です。新規登録は拡張機能から行ってください。」
   - ボタン: 「拡張機能をインストール」（Chrome Web Storeへリンク）
   - 補足リンク: 「GitHubからダウンロード」「導入ガイドを見る」

### バックエンド（Professor）との連携
- Professor API: `POST /auth/login` が `user_not_found` を返した場合
- フロントエンド: 拡張機能誘導画面へルーティング
- エラーコード: `AUTH_USER_NOT_REGISTERED`（`ERROR_CODES.md`に追加）

---

## まず読む（最短ルート）（続き）
3. FSD 全体像：`01_architecture/FSD_OVERVIEW.md`
4. レイヤー境界とバックエンド対応：`01_architecture/FSD_LAYERS.md`
5. Slices とバックエンド境界の対応：`01_architecture/SLICES_MAP.md`
6. 認証・セッション管理：`03_integration/AUTH_SESSION.md` ← **Phase 2以降の必読**
7. データ取得の契約（DAL）：`01_architecture/DATA_ACCESS_LAYER.md`
8. API 契約運用（バックエンドとの通信）：`03_integration/API_CONTRACT_WORKFLOW.md`
9. API 生成（Orval）：`03_integration/API_GEN.md`
10. バージョニング/廃止：`03_integration/API_VERSIONING_DEPRECATION.md`
11. エラー処理の標準：
   - `03_integration/ERROR_HANDLING.md`
   - `03_integration/ERROR_CODES.md`
12. キャッシュ/再検証：`01_architecture/CACHING_STRATEGY.md`
13. セキュリティ（CSP/ヘッダー）：`03_integration/SECURITY_CSP.md`
14. 運用（最小）：
    - `05_operations/OBSERVABILITY.md`
    - `05_operations/RELEASE.md`
    - `05_operations/PERFORMANCE.md`

---

## Architecture
- FSD：
  - `01_architecture/FSD_OVERVIEW.md`
  - `01_architecture/FSD_LAYERS.md`
  - `01_architecture/SLICES_MAP.md`
- Data Access / Cache：
  - `01_architecture/DATA_ACCESS_LAYER.md`
  - `01_architecture/CACHING_STRATEGY.md`
- UI設計：`01_architecture/COMPONENT_ARCHITECTURE.md`
- A11y（最小契約）：`01_architecture/ACCESSIBILITY.md`
- FSD ツール運用：`01_architecture/FSD_TOOLING.md`
- レジリエンス（FE版）：`01_architecture/RESILIENCY.md`

---

## Tech Stack
- `02_tech_stack/STACK.md`
- `02_tech_stack/MUI_PIGMENT.md`
- `02_tech_stack/SSR_HYDRATION.md`
- `02_tech_stack/STATE_QUERY.md`
- `02_tech_stack/SERVER_ACTIONS.md`
- `02_tech_stack/ROUTING_UX_CONVENTIONS.md`

---

## Integration（契約/境界）
- API 生成：`03_integration/API_GEN.md`
- API 契約ワークフロー：`03_integration/API_CONTRACT_WORKFLOW.md`
- バージョニング/廃止：`03_integration/API_VERSIONING_DEPRECATION.md`
- エラー形式/扱い：`03_integration/ERROR_HANDLING.md`
- エラーコード：`03_integration/ERROR_CODES.md`
- CSP/ヘッダー：`03_integration/SECURITY_CSP.md`
- Auth/Session：`03_integration/AUTH_SESSION.md`
- i18n/Locale（必要な場合）：`03_integration/I18N_LOCALE.md`
- Docker 環境：`03_integration/DOCKER_ENV.md`

---

## Testing
- 戦略：`04_testing/TEST_STRATEGY.md`
- ピラミッド：`04_testing/TEST_PYRAMID.md`
- 性能（フロント）：`04_testing/PERFORMANCE_LOAD_TESTING.md`

---

## Operations
- 観測性：`05_operations/OBSERVABILITY.md`
- 性能：`05_operations/PERFORMANCE.md`
- リリース：`05_operations/RELEASE.md`
- CI/CD：`05_operations/CI_CD.md`
- SLO/アラート：`05_operations/SLO_ALERTING.md`
- Secrets/Key：`05_operations/SECRETS_KEY_MANAGEMENT.md`
- Identity/Zero Trust：`05_operations/IDENTITY_ZERO_TRUST.md`
- 脆弱性運用：`05_operations/VULNERABILITY_MANAGEMENT.md`
- サプライチェーン：`05_operations/SUPPLY_CHAIN_SECURITY.md`
- インシデント：`05_operations/INCIDENT_POSTMORTEM.md`

---

## Requirements（要件管理）
- ポータル：`06_requirements/README.md`
- ページ要件：`06_requirements/pages/`
- コンポーネント要件：`06_requirements/components/`

---

## Skills（Agent向け：短い実務ルール）
- `skills/README.md`

運用の基本：
- "迷ったらコードではなくドキュメントを更新して契約を変える"
- "例外は増やさず、境界の切り方を見直す"

---

## バックエンドドキュメントとの関係

���ロントエンドは **Professor（Go）** を通じてバックエンドと通信します。

- バックエンド全体の責務と契約：`../eduanimaR_Professor/docs/README.md`
- バックエンドとフロントエンドの対応関係：`01_architecture/FSD_LAYERS.md` 内の対応表を参照
- API契約の詳細：`03_integration/API_CONTRACT_WORKFLOW.md` および `../eduanimaR_Professor/docs/03_integration/API_GEN.md`