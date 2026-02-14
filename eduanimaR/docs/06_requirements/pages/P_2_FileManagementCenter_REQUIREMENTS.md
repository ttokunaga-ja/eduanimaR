# Page Requirements

## Meta
- Page ID: P_2
- Page Name: File Management Center（資料管理センター）
- Route (App Router): `/app/materials`
- FSD placement (expected): `src/pages/file-management-center`
- Status: approved
- Last updated: 2026-02-15

---

## 1) Goal / Outcome
- 科目横断で資料を整理できる（検索、状態確認、一括操作）
- ingest 失敗等の管理作業をこの画面に集約し、日常のチャット体験を軽くする

成功の定義：
- 資料一覧がテーブルで表示され、検索/削除/状態確認が行える

## 2) Non-goals
- 資料の内容閲覧を主目的にしない（必要ならプレビューへ誘導）
- Q&A の実行（チャットページで行う）

---

## 3) Primary Users / Permissions
- 対象：ログイン済み個人ユーザー
- 認証が必要か：Yes
- 401：再ログイン
- 403：アクセス不可

関連：
- Auth/Session：`../../03_integration/AUTH_SESSION.md`

---

## 4) User Stories
- As a student, I want to find a file across subjects, so that I can clean up quickly.
- As a student, I want to see ingestion status, so that I know what’s ready for Q&A.
- As a student, I want to delete unnecessary files in bulk, so that my library stays clean.

---

## 5) UI Structure（情報設計）

- Top Global Bar（固定、科目セレクタは表示するが、このページでは「横断」が主）
- Main Content（全画面管理ビュー）
  - 検索（キーワード、科目フィルタ）
  - データグリッド（テーブル）
    - 列例：科目、ファイル名、タイプ、アップロード日時、ingest 状態、操作
  - 一括操作（削除など）

---

## 6) States（Must）

- Loading state:
  - テーブル skeleton

- Empty state:
  - 検索結果0件：「一致する資料がありません」
  - 全体0件：「資料がありません」+ チャットページのアップロード導線

- Error state:
  - 401/403：分類通り
  - Upstream：再試行

- Success state:
  - テーブルに資料が表示され、操作できる

---

## 7) Data Requirements

### 7.1 Queries (Read)
- 科目横断の materials 一覧（ingest 状態、メタデータ）

注意：現時点の Professor OpenAPI には Read API が未定義のため、契約追加が必要。

### 7.2 Mutations (Write)
- Delete materials（将来）
- Retry ingest（将来）

注意：これらも OpenAPI に未定義。Contract First で追加する。

---

## 8) Forms（該当する場合）

- 検索条件入力（キーワード等）
- 一括削除は確認（取り消し不可の場合）

---

## 9) i18n（Must）

- namespace（例）：`materialsCenter`, `errors`

---

## 10) Accessibility（Must）

- テーブルはキーボード操作可能（行選択/チェックボックス）
- 検索入力はラベル付与
- 一括操作はフォーカス順序を壊さない

---

## 11) Observability

- `materials_center_viewed`
- `materials_search_applied`
- `materials_bulk_delete_clicked` / `materials_bulk_delete_succeeded` / `materials_bulk_delete_failed`

---

## 12) Performance Notes

- テーブルはページング or 仮想化（件数増加時）

---

## 13) Acceptance Criteria（Must）

- [ ] 科目横断で資料がテーブル表示される
- [ ] キーワード検索で絞り込める
- [ ] ingest 状態が一覧で分かる

---

## 14) Open Questions

- materials Read API（一覧/検索）の OpenAPI 追加内容
- 一括削除/リトライの契約（権限・監査）
