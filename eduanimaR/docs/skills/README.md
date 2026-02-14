# Skills（Frontend / FSD Template）

この `skills/` は、長いドキュメント（docs本体）の要点を **短い実務ルール（Must/禁止/チェックリスト）** に圧縮したものです。

目的：
- AI/人間が「毎回同じ判断」をできるようにする
- 変更頻度が高い/破壊的変更が入りやすい領域の事故を減らす

注意：本リポジトリは 2026 年時点の運用を意図しますが、モデルの知識は固定です。
そのため Skill は「最新仕様の丸暗記」ではなく、**変化に強い判断軸（境界/禁止/確認手順）** を中心に書きます。

---

## 最新版の確認（2026-02-11 時点）

このテンプレでは特定プロジェクトの依存を同梱していないため、最新版は外部ソース（npm / nodejs.org）で都度確認します。

取得元：
- npm：`npm view <package> version`
- Node：`curl -fsSL https://nodejs.org/dist/index.json`

最新版（dist-tag: latest、2026-02-11に取得）：

| Tech | Package | Latest |
| --- | --- | --- |
| Next.js | `next` | `16.1.6` |
| React | `react` / `react-dom` | `19.2.4` / `19.2.4` |
| TypeScript | `typescript` | `5.9.3` |
| MUI | `@mui/material` | `7.3.7` |
| Pigment | `@pigment-css/react` | `0.0.30` |
| TanStack Query | `@tanstack/react-query` | `5.90.20` |
| Orval | `orval` | `8.2.0` |
| Zod | `zod` | `4.3.6` |
| React Hook Form | `react-hook-form` | `7.71.1` |
| Vitest | `vitest` | `4.0.18` |
| Playwright | `@playwright/test` | `1.58.2` |
| ESLint | `eslint` | `10.0.0` |
| Boundaries | `eslint-plugin-boundaries` | `5.4.0` |

Node（公式 index.json、2026-02-11に取得）：
- latest LTS：`v24.13.1`（Krypton）
- latest Current：`v25.6.1`

---

## 読む順（最短）
1. `SKILL_NEXTJS_APP_ROUTER.md`
2. `SKILL_NEXTJS_TURBOPACK.md`
3. `SKILL_MUI_PIGMENT_CSS.md`
4. `SKILL_TANSTACK_QUERY.md`
5. `SKILL_ORVAL_OPENAPI.md`
6. `SKILL_ESLINT_BOUNDARIES.md`
7. `SKILL_FSD_BOUNDARIES.md`（FSD境界強制）⭐ 新規
8. `SKILL_SSE_CLIENT.md`（SSEストリーミング）⭐ 新規
9. `SKILL_TAILWIND_CSS.md`（Web/拡張でのスコープ制御）⭐ 新規
10. `SKILL_TESTING_VITEST.md`
11. `SKILL_TESTING_PLAYWRIGHT.md`
12. `SKILL_TYPESCRIPT.md`
13. `SKILL_NODE_DOCKER_RUNTIME.md`
14. `SKILL_ZOD_RHF_FORMS.md`

---

## 共通の運用原則（Must）
- 迷ったら、実装より先に docs（契約）を更新する
- deep import をしない（Public API）
- “例外追加” で逃げず、構造（境界/責務）を直す
