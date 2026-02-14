# Routing UX Conventions（loading / error / not-found / streaming）

このドキュメントは、Next.js App Router の特殊ファイル（`loading.tsx`, `error.tsx`, `not-found.tsx`, `global-error.tsx`）を
どの粒度で置き、どの種類の失敗/待ちをどこで扱うかを固定して、UX と運用の一貫性を担保します。

関連：
- エラーの標準：`../03_integration/ERROR_HANDLING.md`
- 観測性：`../05_operations/OBSERVABILITY.md`

---

## 結論（Must）

- ルートセグメントごとに “意味のある” `loading.tsx` を用意する（白画面を作らない）
- 予期しない例外は `error.tsx`（Error Boundary）で受け、観測に載せる
- 404 は `notFound()` + `not-found.tsx` を使う（文字列分岐で 404 を作らない）
- UIの文言は翻訳キーとして管理し、表示内容は各言語の JSON から読み出すこと（not-found や error の表示文言もハードコーディングしない）。文字列を条件分岐に使うことは避ける（詳細：`../03_integration/I18N_LOCALE.md`）
- `global-error.tsx` は最後の砦（最小）

---

## 1) loading.tsx（Must）

- `loading.tsx` は 해당セグメント配下の `page.tsx` を自動で Suspense で包む
- “意味のある skeleton” を返す（レイアウトが崩れない最低限の形）
- data fetching を `loading.tsx` で行わない（UI のみ）

チェックリスト：
- [ ] 主要導線は常に何か表示される
- [ ] 連続遷移でも UI が崩れない

---

## 2) error.tsx（Must）

- `error.tsx` は Client Component（`"use client"` 必須）
- `reset()` を提供し、可能なら “再試行” を提供する
- logging は `useEffect` で行い、運用へ送る（console だけで終えない）

粒度：
- 重要なセグメント（例：`(routes)/app/history`）には局所 `error.tsx` を置き、全体巻き込みを防ぐ

---

## 3) global-error.tsx（Must when needed）

- root の例外 UI。
- `global-error.tsx` は root layout を置き換えるため **`<html>` と `<body>` が必要**
- 基本は最小 UI（謝罪 + リロード）に留め、詳細な分類はしない

---

## 4) not-found.tsx（Must）

- “対象がない” は `notFound()` を使う
- 404 を throw で表現しない（読みづらくなる）

---

## 5) Streaming の注意（Should）

- streaming 中は HTTP status が 200 のまま返ることがある（soft 404 の文脈）
- compliance / analytics 的に 404 status が必要なら、streaming 前に存在確認する（proxy などで高速に）

---

## 禁止（AI/人間共通）

- エラー/404 を message 文字列比較で分岐
- 例外を握りつぶして fallback だけ返し、観測に載せない
- `loading.tsx` を置かずに初期表示が空になる導線を放置
