# FSD Tooling（lint / generators / monorepo）

このドキュメントは、FSD の構造を “人間の善意” に依存させず、
ツールで継続的に検証・生成する運用を固定します。

関連：
- レイヤー契約：`FSD_LAYERS.md`
- slices：`SLICES_MAP.md`

---

## 結論（Must）

- 境界違反は ESLint（`eslint-plugin-boundaries`）で機械的に落とす
- slices の追加/変更は `SLICES_MAP.md` に反映し、レビュー対象にする
- 大規模化する場合は monorepo（packages）でも FSD を維持できる前提で設計する

---

## 1) Lint（Must）

- `eslint-plugin-boundaries` を SSOT とする
- ルール例：
  - 上位レイヤー → 下位レイヤーのみ import 可
  - 同一レイヤーの別 slice を import 禁止

---

## 2) Generators（Should）

- slice の雛形は generator（CLI/IDE）で揃えると、構造逸脱が減る
- 導入する場合、生成物の “Public API ファイル” を必ず作る（import を固定）

---

## 3) Monorepo（Should）

- packages に分割しても、各 package 内で FSD を適用できる
- cross-package import は “公共APIのみ” に制限する（境界を守る）

---

## 禁止（AI/人間共通）

- boundaries 違反を例外で通し続ける（例外が常態化する）
- SLICES_MAP を更新せずに slice を増やす
