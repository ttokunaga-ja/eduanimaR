# SKILL: TanStack Query（v5/v6）

対象：サーバー状態（API）をキャッシュし、UIの一貫性を保つ。

変化に敏感な領域：
- SSR/Hydration との統合
- invalidate と Next 再検証の二重管理

関連：
- `../02_tech_stack/STATE_QUERY.md`
- `../01_architecture/CACHING_STRATEGY.md`
- `../02_tech_stack/SSR_HYDRATION.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `@tanstack/react-query`: `5.90.20`

---

## Must
- “RSCが主” のデータは Next の再検証で整合させる
- “Client hooksが主” のデータは Query を主にする
- 二重管理（両方でなんとなくinvalidate）をしない

### 実装メモ（事故りやすい点）

- mutation 後に「Nextの再検証」と「Query invalidate」を混ぜると整合が破綻しやすい
- どちらをSSOTにするかを `CACHING_STRATEGY.md` とセットで決める

## 禁止
- mutation 後の整合を画面ごとに場当たりで直す

## チェックリスト
- [ ] このデータは RSC 主か Client 主か？
- [ ] 整合の責務は Next か Query のどちらに置くか？
- [ ] エラー分類（再試行可否）が揃っているか？
