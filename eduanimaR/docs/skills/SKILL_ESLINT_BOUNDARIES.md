# SKILL: ESLint + eslint-plugin-boundaries（FSD境界強制）

対象：FSD のレイヤー/スライス境界を “人の善意” ではなくツールで強制する。

変化に敏感な領域：
- ESLint の設定形式（flat config等）
- import alias / path 解決

関連：
- `../01_architecture/FSD_LAYERS.md`
- `../05_operations/CI_CD.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `eslint`: `10.0.0`
- `eslint-plugin-boundaries`: `5.4.0`
- `eslint-plugin-import`: `2.32.0`
- `@typescript-eslint/parser`: `8.55.0`
- `@typescript-eslint/eslint-plugin`: `8.55.0`

---

## Must
- 境界違反はCIで落とす（例外を増やさない）
- deep import を禁止し、Public API を守る

### 実装メモ（テンプレの形）

- Flat config（`eslint.config.mjs`）で boundaries の `elements` と依存方向を固定する
- `no-restricted-imports` で deep import を止める

## 禁止
- "一時しのぎ" の例外ルール追加
- テストだけ境界を緩める（抜け道になる）

## チェックリスト
- [ ] import 方向は単方向（`app→...→shared`）か？
- [ ] 同一レイヤー横断（`features→features`）が増えていないか？
- [ ] 境界違反を設定で無効化していないか？
