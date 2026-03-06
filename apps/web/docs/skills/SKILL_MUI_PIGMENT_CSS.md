# SKILL: MUI v6 + Pigment CSS（Zero-runtime CSS）

対象：ランタイムCSS計算を避け、性能（Vitals）を守る。

変化に敏感な領域：
- Pigment CSS の仕様/制約（成熟途上になりやすい）
- `sx` / 動的スタイルの扱い

関連：
- `../02_tech_stack/MUI_PIGMENT.md`
- `../05_operations/PERFORMANCE.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `@mui/material`: `7.3.7`
- `@mui/system`: `7.3.7`
- `@mui/icons-material`: `7.3.7`
- `@mui/material-pigment-css`: `7.3.7`
- `@pigment-css/react`: `0.0.30`

（確認：`npm view @mui/material version` など）

---

## Must
- 動的なスタイル（`sx` 等）の使用は `shared/ui` のラッパーに寄せる
- “動くからOK” ではなく、ビルド時CSS生成の前提を守る

### 実装メモ（方針）

- `sx` をアプリ全体に散らすと「性能/一貫性/アップグレード耐性」が落ちる
- “アプリ側は primitives を使うだけ” に寄せ、`shared/ui` で吸収する

## 禁止
- Emotion など別ランタイムCSSの混入（方針逸脱）
- feature/pages 側で `sx` を乱用して設計を崩す

## チェックリスト
- [ ] `sx` の利用は `shared/ui` に閉じているか？
- [ ] 追加したUIが hydration コストを増やしていないか？
- [ ] Pigmentの制約に反する書き方をしていないか？
