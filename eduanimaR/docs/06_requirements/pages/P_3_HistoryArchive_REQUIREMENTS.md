# Page Requirements

## Meta
- Page ID: P_3
- Page Name: History Archive（ラーニング・アーカイブ）
- Route (App Router): `/app/history`
- FSD placement (expected): `src/pages/history-archive`
- Status: approved
- Last updated: 2026-02-15

---

## 1) Goal / Outcome
- 全期間の質問履歴を俯瞰し、試験前に「つまずき」を再確認できる
- キーワードで過去の質問を検索し、当時の文脈へ戻れる

成功の定義：
- 履歴が一覧化され、検索で絞り込め、スレッド詳細へ遷移できる

## 2) Non-goals
- チャット実行（チャットページで行う）
- 科目/資料の管理操作（資料管理センターで行う）

---

## 3) Primary Users / Permissions
- 対象：ログイン済み個人ユーザー
- 認証が必要か：Yes
- 401：再ログイン
- 403：アクセス不可

---

## 4) User Stories
- As a student, I want to browse my past questions by date, so that I can review efficiently.
- As a student, I want to search by keyword, so that I can find my previous confusion quickly.

---

## 5) UI Structure（情報設計）

- Top Global Bar（固定）
- Main Content（全画面管理ビュー）
  - 表示方式：カレンダービュー または 無限スクロールリスト（いずれかを採用）
  - キーワード検索
  - スレッドカード（タイトル、日付、科目、要約）

---

## 6) States（Must）

- Loading state:
  - リスト skeleton

- Empty state:
  - 履歴0件：「履歴がありません」+ チャットへ戻る導線
  - 検索0件：「一致する履歴がありません」

- Error state:
  - 401/403/Upstream：分類通り

- Success state:
  - 履歴が閲覧でき、対象スレッドへ移動できる

---

## 7) Data Requirements

### 7.1 Queries (Read)
- 履歴一覧（期間、ページング）
- キーワード検索

注意：Professor OpenAPI に未定義のため契約追加が必要。

### 7.2 Mutations (Write)
- 履歴の削除（将来）

---

## 8) Forms（該当する場合）

- 検索入力（キーワード）

---

## 9) i18n（Must）

- namespace（例）：`historyArchive`, `errors`

---

## 10) Accessibility（Must）

- リストはキーボード操作可能
- 検索入力はラベル付与
- 無限スクロールの場合、読み上げ/フォーカスが破綻しないようにする（必要なら「さらに読み込む」ボタン方式）

---

## 11) Observability

- `history_archive_viewed`
- `history_search_applied`
- `history_thread_opened`

---

## 12) Performance Notes

- 履歴はページング前提（無限スクロールでも chunk 単位で取得）

---

## 13) Acceptance Criteria（Must）

- [ ] 全期間の履歴が閲覧できる
- [ ] キーワード検索で絞り込める
- [ ] スレッドを開いて当時の内容へ戻れる

---

## 14) Open Questions

- 履歴表示を「カレンダー」か「リスト」どちらで確定するか
- 履歴 Read API（一覧/検索）の OpenAPI 追加内容
