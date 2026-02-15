# Auth & Session Policy（Cookie / CSRF / Cache）

Last-updated: 2026-02-15

このドキュメントは、Next.js（App Router）を BFF として運用する際の
認証・セッション・Cookie とキャッシュの相互作用を “契約” として固定します。

関連：
- DAL：`../01_architecture/DATA_ACCESS_LAYER.md`
- キャッシュ：`../01_architecture/CACHING_STRATEGY.md`
- セキュリティヘッダー/CSP：`SECURITY_CSP.md`

---

## 結論（Must）

- セッションは **HttpOnly Cookie** を基本（アクセストークンを LocalStorage に置かない）
- CSRF 方針（SameSite / Origin check / token）をプロジェクトとして固定する
- ユーザー依存データは “意図せず共有キャッシュ” されないようにする（動的化 or no-store か設計で担保）
- 認証状態の変更（login/logout）は UI に即反映されるよう、再検証と境界を明示する

---

## 1) Cookie ポリシー（テンプレ）

- `HttpOnly`: true（JS から読めない）
- `Secure`: true（本番）
- `SameSite`: `Lax`（基本）
- `Path`: `/`
- `Domain`: 必要時のみ

注意：Cookie 設計は CSRF/CORS/サブドメイン構成に依存するため、プロジェクト固有で確定させる。

---

## 2) CSRF（Must）

- Server Actions は UI 起点の mutation で第一選択（origin チェック等の枠組みがある）
- Route Handler を公開 API として使う場合：
  - CORS を明示（許可 origin を最小化）
  - Cookie 認証の場合は CSRF 対策を必ず入れる（token / double submit 等、方針を決める）

---

## 3) Cache との相互作用（Must）

- `cookies()` / `headers()` などの Dynamic API はルートを動的化し、Full Route Cache に影響する
- セッション Cookie の set/delete は “認証状態が変わる” ため、UI の整合性を崩しやすい
- ユーザー依存データの fetch では、キャッシュ設計を docs に残す（暗黙禁止）

運用上の目安：
- 認証必須ページ：動的化（意図して）
- 公開ページ：静的 + ISR / tag revalidate

---

## 4) 実装責務の分離（Must）

- 認可（Authorization）は DAL に閉じ込める
- UI は認証/認可の “分類結果” を受けて分岐する（code で）

---

## 禁止（AI/人間共通）

- トークンを LocalStorage に保存
- ユーザー依存レスポンスを “共通キャッシュ” に載せる
- 例外を握りつぶして「ログインし直して」で済ませる（分類して扱う）

---

## eduanimaR固有の認証フロー

### Phase 1（ローカル開発）
- 固定の `dev-user` を使用
- フロントエンドに認証UIは存在しない
- Professor APIが開発モードでdev-userを自動設定

### Phase 2以降（本番）
- **SSO認証**: Google / Meta / Microsoft / LINE
- **ユーザー登録フロー**:
  1. ユーザーがChrome拡張機能をインストール
  2. LMS上でSSO認証
  3. Professorがユーザー登録・科目同期
  4. Moodle資料の自動検知・アップロード開始

### Web版の認証制限
- **新規登録禁止**: Web版からの新規ユーザー登録は無効化
- **ログインのみ許可**: 拡張機能でSSO登録したユーザーのみWeb版にログイン可能
- **目的**: 大画面での閲覧・履歴確認専用
- **実装方針**: フロントエンドに開発専用の認証UIを実装してはならない

### セッション管理
- Cookie-based session（Next.js middleware）
- セッション期限: 7日間（更新可能）
- セッションストア: Redis（Phase 2以降）
