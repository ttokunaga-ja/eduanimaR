# Requirements（Pages / Components）

この `06_requirements/` は、プロダクト要件（何を作るか）を **SSOT（Single Source of Truth）** として固定する場所です。

目的：
- AI / 人間が「仕様の穴埋め」をしない
- 実装・テスト・観測の判断基準を先に決める
- FSD の配置（pages/widgets/features…）と要件をつなぐ

関連（既存の契約）：
- ルーティング/UX：`../02_tech_stack/ROUTING_UX_CONVENTIONS.md`
- エラーの標準：`../03_integration/ERROR_HANDLING.md`
- エラーコード：`../03_integration/ERROR_CODES.md`
- データ取得と境界：`../01_architecture/DATA_ACCESS_LAYER.md`
- キャッシュ：`../01_architecture/CACHING_STRATEGY.md`
- A11y：`../01_architecture/ACCESSIBILITY.md`
- 観測性：`../05_operations/OBSERVABILITY.md`

---

## 結論（Must）

- 新しい画面（route）を作る前に、ページ要件を `pages/` に作成する
- 再利用前提の UI ブロック/コンポーネントを作る前に、コンポーネント要件を `components/` に作成する
- 要件には必ず「成功条件（Acceptance Criteria）」と「状態（loading / empty / error）」を含める
- 不明点は推測で埋めず、要件側へ追記して“契約”を更新する

---

## ディレクトリ

- `pages/`：ページ（URL/route）単位の要件
- `components/`：コンポーネント単位の要件（共通部品・複合部品）

---

## 命名規約（推奨）

ページ要件：
- `pages/P_1_<PageName>_REQUIREMENTS.md`
- 例：`pages/P_1_HomeSearchPage_REQUIREMENTS.md`

コンポーネント要件：
- `components/C_1_<ComponentName>_REQUIREMENTS.md`
- 例：`components/C_1_HomeSearchPanel_REQUIREMENTS.md`

注意：番号は並び替えと参照の安定のための“ID”として扱い、途中挿入は末尾番号で追加します。

---

## テンプレ

- ページ要件テンプレ：`pages/PAGE_REQUIREMENTS_TEMPLATE.md`
- コンポーネント要件テンプレ：`components/COMPONENT_REQUIREMENTS_TEMPLATE.md`

---

## FSD への落とし込み（目安）

- Page requirements → `src/pages/<slice>/ui/Page.tsx` を中心に合成
- Component requirements → 再利用粒度に応じて `shared/ui` / `widgets/*/ui` / `features/*/ui` へ配置

要件に「想定する置き場（FSD layer/slice）」を書いておくと、境界違反や作り直しが減ります。
