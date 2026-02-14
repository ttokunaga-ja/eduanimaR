# Handbook — このリポジトリのテンプレート説明

> この README は「各ページをテンプレート化」して運用するための案内です。

---

## 使い方（短く）
- 各ファイルは既存のドキュメントをテンプレート形式に書き換え済みです。
- 各ページを編集するときは、ファイル上部の `Title`/`Owner`/`Status`/`Last-updated` を必ず更新してください。
- 重大な方針変更は PR → 主要メンバーのレビューを経てマージします。

---

## 共通フロントマター（必須メタ）
```
Title: <ページタイトル>
Description: <1行の要約>
Owner: @github-handle
Reviewers: @reviewer1, @reviewer2
Status: Draft | Published | Archived
Last-updated: YYYY-MM-DD
Tags: tag1, tag2
```

---

## 目次（テンプレ化済みファイル）
- `01_philosophy/MISSION_VALUES.md` — ミッション／バリューのテンプレ
- `01_philosophy/PRIVACY_POLICY.md` — プライバシー方針テンプレ
- `01_philosophy/TERMS_OF_SERVICE.md` — 利用規約テンプレ
- `02_strategy/LEAN_CANVAS.md` — リーンキャンバステンプレ
- `03_customer/PERSONAS.md` / `03_customer/CUSTOMER_JOURNEY.md` — ペルソナ／ジャーニー
- `04_product/*` — Roadmap、Brand、Playbook、Policy などのテンプレ
- `05_goals/OKR_KPI.md` — OKR/KPI テンプレ

---

## 更新ルール（再掲）
1. まず `Owner` を更新 → 内容を埋める → PR を作成
2. 少なくとも1名の `Reviewer` にレビューを依頼
3. マージ後、`Last-updated` を更新

---

## 補足（ローカル運用）
- ドキュメントは四半期ごとに見直してください。
- 機密・法務に関わる変更は必ず法務レビューを通してください。

(テンプレ適用日: YYYY-MM-DD)