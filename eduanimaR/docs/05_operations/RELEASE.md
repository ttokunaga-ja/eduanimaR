# Release & Environments（環境/リリース/ロールバック）

このドキュメントは、環境差分とリリース手順を固定し、
「本番だけ壊れる」「戻せない」を防ぐための運用契約です。

---

## 結論（Must）

- 環境差分は `.env` とドキュメントに集約（コード内ハードコード禁止）

テンプレのデフォルト：
- `.env.example` をベースに `.env.local`（local）/ プラットフォームの環境変数（staging/production）へ反映する
- リリースは “戻せる” こと（ロールバック手順）を前提に設計する
- DB/API の互換性を考慮し、段階的リリースを可能にする（Feature Flag など）

---

## 環境一覧（プロジェクト固有で埋める）

- local
- staging
- production

各環境での差分：
- API base URL
- 認証方式（Cookie/Bearer）
- 外部連携（Analytics / Error reporting）

---

## 事前確認（Must）

- `next build` が通る
- 主要ページの SSR/Hydration が崩れていない（初期表示/操作）
- 主要 mutation 後の整合（キャッシュ無効化）が正しい

---

## ロールバック

- ロールバック条件（SLO 逸脱、致命バグ、決済影響など）を明文化
- DB マイグレーションが絡む場合は、
  - 後方互換
  - 二段階デプロイ
  - もしくは “戻せない” ことの合意
を必ず行う
