# Page Requirements

## Meta
- Page ID: P_1
- Page Name: Chat Workspace（Hybrid Tab Layout）
- Route (App Router): `/app`
- FSD placement (expected): `src/pages/chat-workspace`
- Status: approved
- Last updated: 2026-02-15

---

## 1) Goal / Outcome
- 科目（subject）スコープを明示した状態で、質問→回答（SSE）を最短で回せる
- 回答は必ず Source（根拠）を提示し、ユーザーが即座に原典へ戻れる

成功の定義：
- `subject_id` を伴う質問が送信され、SSEで回答がストリーミング表示される
- 回答に含まれる Source をクリックして確認できる

## 2) Non-goals
- 科目横断の大規模整理（それは管理ページで行う）
- 履歴の全文検索（それは History Archive で行う）

---

## 3) Primary Users / Permissions
- 対象：SSOでログイン済みの個人ユーザー
- 認証が必要か：Yes
- 未ログイン時：ログイン導線（SSO）を表示し、質問/アップロードは不可
- 権限不足（403）：アクセス不可の表示（問い合わせ導線は残す）

関連：
- Auth/Session：`../../03_integration/AUTH_SESSION.md`

---

## 4) User Stories
- As a student, I want to select a subject, so that my questions search only that subject’s materials.
- As a student, I want to add or delete a subject from the subject dropdown, so that I can manage my subjects with minimal effort.
- As a student, I want streaming answers with sources, so that I can verify quickly.
- As a student, I want quick access to recent threads and files, so that I can continue work without leaving chat.

---

## 5) UI Structure（情報設計）

このページは **「対話（Chat）」と「管理（Management）」を分離**した SPA 風のアプリシェルであり、デフォルト表示は Chat。

上から順：

1. Top Global Bar（固定ヘッダー / widget）
   - 左：Hamburger（Sidebar開閉）
  - 中央：Subject Context Selector（科目選択ドロップダウン）
    - 科目の切替
    - 科目の新規追加（この場で作成）
    - 科目の削除（確認あり）
   - 右：
     - お問い合わせ（外部 Google Form へ新規タブ）
    - ログアウトボタン

2. Tabbed Sidebar（引き出し式 / widget）
   - タブ：`履歴` / `資料`
   - `履歴`：直近スレッド一覧 + 下端固定リンク「すべての履歴を見る ↗」（History Archive へ遷移）
   - `資料`：当該科目のファイル一覧 + ingest 状態（解析中/完了） + 下端固定リンク「資料管理センターを開く ↗」（File Management Center へ遷移）
   - 最下部：手動アップロードボタン

3. Main Content（流動的エリア / page）
   - Chat message area
   - Sticky bottom input（送信/送信中表示）
   - Reference Card（Source をカード表示、クリックでプレビュー/リンク）

---

## 6) States（Must）

- Loading state:
  - 初期：ヘッダーと空のチャット骨格を表示
  - Sidebar：履歴/資料タブのリストは skeleton
  - SSE：接続中は「生成中/接続中」を表示（ストリーミングUI）

- Empty state:
  - 科目未選択：入力欄の上に「科目を選択してください」導線
  - 履歴0件：`履歴` タブに「まだ履歴がありません」+ まず質問する導線
  - 資料0件：`資料` タブに「資料がありません」+ 手動アップロード導線

- Error state:
  - 401：再ログイン導線
  - 403：権限なし表示
  - 404（request_id不整合/期限切れなど）：再送（質問をやり直す）導線
  - 429：待機→再送（UIで抑制）
  - 5xx/UPSTREAM：再試行ボタン + 問い合わせ導線
  - 科目の作成/削除の失敗：
    - validation / conflict / forbidden / not found を想定
    - 失敗時は現在の科目コンテキストを保持し、再試行またはキャンセル導線
  - 表示文言は翻訳キー（例：`errors.unauthorized`, `errors.upstreamTimeout`）

- Success state:
  - チャットが継続でき、ソースカードが表示される

関連：
- Routing UX conventions：`../../02_tech_stack/ROUTING_UX_CONVENTIONS.md`
- Error handling：`../../03_integration/ERROR_HANDLING.md`
- Error codes：`../../03_integration/ERROR_CODES.md`

---

## 7) Data Requirements

### 7.1 Queries (Read)
- Subject list（科目一覧）
- Current subject context（選択中科目）
- Recent threads list（当該科目）
- Materials list（当該科目、ingest status 含む）

取得方法：
- 認証/権限を含むため、原則 Client + TanStack Query（ただし初期描画要件に応じてRSCでも可）

キャッシュ方針（暫定）：
- `subject_id` 切替でクエリキーを分離
- 履歴/資料は stale-while-revalidate（短め）

注意：Professor OpenAPI（SSOT）に Read 系の一覧 API が未定義のため、契約追加が必要。
同様に、科目の Create/Delete API も SSOT に追加が必要。

### 7.2 Mutations (Write)
- Create subject（新規科目作成）
  - 成功後：subject list を invalidate/refetch、作成した subject を選択状態にする
- Delete subject（科目削除）
  - 成功後：subject list を invalidate/refetch
  - 削除した subject が選択中の場合：未選択 or 次候補へ切替し、Sidebar/Chat のスコープを更新
- Start question：`POST /v1/questions`（必須：`subject_id`, `question`）
  - 成功後：`events_url` で SSE 購読開始
- Upload material：`POST /v1/materials`（必須：`subject_id`, `file`）
  - 成功後：当該科目の materials を invalidate/refetch

関連：
- DAL：`../../01_architecture/DATA_ACCESS_LAYER.md`
- Cache：`../../01_architecture/CACHING_STRATEGY.md`
- Server state：`../../02_tech_stack/STATE_QUERY.md`

---

## 8) Forms（該当する場合）

- Chat input
  - 必須：Yes（空送信不可）
  - 送信中：ボタン disabled + スピナー
  - 送信失敗：再試行

- Manual upload
  - 必須：ファイル + `subject_id`
  - validation：サイズ上限、拡張子（許容形式）

---

## 9) i18n（Must）

- namespace（例）：`appShell`, `chat`, `sidebar`, `errors`

関連：`../../03_integration/I18N_LOCALE.md`

---

## 10) Accessibility（Must）

- Hamburger：キーボード操作可（Enter/Space）
- Sidebar tabs：`role="tablist"`/`tab`/`tabpanel` のセマンティクス
- Sidebar 開閉時：フォーカストラップは不要（drawer扱い）だが、開いたら先頭要素へフォーカス
- Subject selector：ラベル/説明を付与し、現在スコープを読み上げ可能
- 新規メッセージ追加時：自動スクロールはユーザー操作を阻害しない（最新へジャンプボタン等は将来）

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 11) Observability

- `subject_changed`（subject_id, from/to）
- `subject_created`（subject_id）
- `subject_deleted`（subject_id, wasCurrent）
- `chat_send_clicked`（subject_id, length）
- `sse_connected` / `sse_reconnect` / `sse_error`
- `upload_clicked` / `upload_succeeded` / `upload_failed`
- `nav_open_history_archive` / `nav_open_file_management`

関連：`../../05_operations/OBSERVABILITY.md`

---

## 12) Performance Notes

- SSE により体感待ち時間を削減
- Sidebar リストは仮想化を検討（件数が増える前は不要）

---

## 13) Acceptance Criteria（Must）

- [ ] 科目未選択では質問送信できず、選択導線が出る
- [ ] 科目選択プルダウンから科目を新規追加でき、追加直後に選択状態になる
- [ ] 科目選択プルダウンから科目を削除でき、削除前に確認が表示される
- [ ] `POST /v1/questions` → SSE で回答がストリーミング表示される
- [ ] 回答の Source がカード形式で表示され、クリック可能
- [ ] Sidebar の「すべての履歴」「資料管理センター」から各ページへ遷移できる

---

## 14) Open Questions

- 科目一覧/履歴一覧/資料一覧の Read API を Professor 側 OpenAPI にどう追加するか
- 科目 Create/Delete API（削除の扱い：cascade/soft delete/制約）を OpenAPI にどう追加するか
- Source プレビューの方式（外部URL遷移/アプリ内プレビュー/権限付きURL）
- アップロード許容形式/サイズ上限
