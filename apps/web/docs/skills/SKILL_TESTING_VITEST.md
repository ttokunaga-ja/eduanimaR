# SKILL: Vitest（Unit/Component/Integration）

対象：速いフィードバックで破壊を止める。

変化に敏感な領域：
- テストが境界違反（deep import）を誘発
- Component test が実装詳細依存になる

関連：
- `../04_testing/TEST_STRATEGY.md`
- `../04_testing/TEST_PYRAMID.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `vitest`: `4.0.18`

---

## Must
- Unit/Component を土台にする（E2E最小）
- テストも Public API 経由で import する

## 禁止
- スナップショットでUI全体固定
- テストのための境界破壊

## チェックリスト
- [ ] 重要な分岐（成功/失敗）を押さえているか？
- [ ] deep import していないか？
- [ ] E2Eに寄せすぎていないか？
