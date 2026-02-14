# SKILL: Turbopack（Next.js Bundler）

対象：開発体験（HMR）を速くしつつ、本番ビルドの再現性を崩さない。

変化に敏感な領域：
- dev と production の差（devで動く≠本番OK）
- bundler 差による挙動差（CSS/loader/edge cases）

関連：
- `../05_operations/CI_CD.md`

---

## Versions（2026-02-11）

Turbopack は Next.js に同梱されるため、実務上は `next` のバージョン固定が重要です。

- `next`: `16.1.6`（dist-tag: latest）

---

## Must
- PR で `next build` を必ず通す（dev server だけで判断しない）
- bundler 依存の回避策は docs 化し、将来の更新に備える

### 実装メモ（運用）

- dev（Turbopack）で動いても、production の `next build` を通して初めて合格
- CSS/生成物（Pigment/Orval）周りは dev/prod 差が出やすいので、CIゲートで止める

## 禁止
- bundler 依存の hack を無記録で入れる

## チェックリスト
- [ ] `next build` はCIで通るか？
- [ ] dev/prod 差で壊れる箇所（CSS生成、dynamic import等）を踏んでいないか？
