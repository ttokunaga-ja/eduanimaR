# Page Requirements Template

## Meta
- Page ID:
- Page Name:
- Route (App Router):
- FSD placement (expected): `src/pages/<slice>`
- Status: draft / approved
- Last updated:

---

## 1) Goal / Outcome
- 何のためのページか（ユーザー価値）
- 成功の定義（例：投稿完了、検索結果に到達、設定が保存される）

## 2) Non-goals
- このページでやらないこと

---

## 3) Primary Users / Permissions
- 対象ユーザー（role/plan/login state）
- 権限（未ログイン時、権限不足時の挙動）
- 認証が必要か：Yes/No

関連：
- Auth/Session：`../../03_integration/AUTH_SESSION.md`

---

## 4) User Stories
- As a ..., I want ..., so that ...

---

## 5) UI Structure（情報設計）
- セクション一覧（上から順）
- 主要コンポーネント（widgets/features/entities の粒度）

---

## 6) States（Must）

各状態で「何を表示するか」を明確化します。

- Loading state:
  - どの範囲を skeleton にするか
  - streaming / suspense の想定
- Empty state:
  - 何を empty とみなすか
  - 次アクション（導線）
- Error state:
  - 想定エラー（ネットワーク、認可、validation、404、500）
  - 表示メッセージ（翻訳キー）
  - リトライ可否
- Success state:

関連：
- Routing UX conventions：`../../02_tech_stack/ROUTING_UX_CONVENTIONS.md`
- Error handling：`../../03_integration/ERROR_HANDLING.md`
- Error codes：`../../03_integration/ERROR_CODES.md`

---

## 7) Data Requirements

### 7.1 Queries (Read)
- 取得するデータ（DTO最小化の前提）
- どこで取得するか（RSC / Client / DAL）
- キャッシュ方針（tag/path, revalidate, no-store の判断）

### 7.2 Mutations (Write)
- 実行する操作
- optimistic update の有無
- 成功後の整合（invalidate / refetch / router refresh）

関連：
- DAL：`../../01_architecture/DATA_ACCESS_LAYER.md`
- Cache：`../../01_architecture/CACHING_STRATEGY.md`
- Server state：`../../02_tech_stack/STATE_QUERY.md`

---

## 8) Forms（該当する場合）
- 入力項目、必須/任意、validation
- 送信中/送信失敗/再送の挙動

---

## 9) i18n（Must）
- 表示文言は翻訳キーで管理する
- ページ固有 namespace（必要なら）

関連：`../../03_integration/I18N_LOCALE.md`

---

## 10) Accessibility（Must）
- キーボード操作
- フォーカス管理（遷移/エラー/モーダル相当UI）
- aria / 見出し構造

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 11) Observability
- 主要イベント（成功/失敗）
- エラー計測（どの失敗を運用に載せるか）

関連：`../../05_operations/OBSERVABILITY.md`

---

## 12) Performance Notes
- SSR/CSR の判断理由
- 画像/リスト/無限スクロールなど負荷点

---

## 13) Acceptance Criteria（Must）
- [ ]
- [ ]
- [ ]

---

## 14) Open Questions
- 未確定事項（判断が必要な点）
