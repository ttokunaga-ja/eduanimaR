# Test Strategy

このドキュメントは、テスト範囲と責務を固定し、AI が過剰な E2E や不適切な層にテストを書かないようにするためのガイドです。

## 採用ツール（固定）
- Unit / Component / Integration: **Vitest**
- E2E: **Playwright**

推奨（事実上の標準）：
- Component testing：React Testing Library（`@testing-library/react`） + `@testing-library/user-event`
- API スタブ：MSW（Orval の MSW 生成も選択肢）

## 目的
- 壊れやすい箇所（仕様・依存境界）を優先してテストする
- FSD の層構造を壊さない（テストが層違反を誘発しない）

## テストの種類（最小）
### Unit
- 対象: `shared/lib`, `entities/*/lib` など純粋関数
- 禁止: DOM を伴う結合テストを Unit に押し込む

### Component
- 対象: `shared/ui`, `entities/*/ui`, `features/*/ui`
- 指針: ユーザー操作と表示を中心に（実装詳細に寄せない）

最低限の観点：
- 主要な表示（ラベル/必須要素）
- 主要な操作（入力→送信→状態変化）
- エラー表示（バリデーション / API失敗）

### Integration（必要な場合のみ）
- 対象: `features` と API Hook の結合
- 方針: API は MSW 等でスタブ（プロジェクト採用に合わせて追記）

Integration を書く基準：
- 重要なユースケースで「APIとUIの接続」が壊れると影響が大きい
- 画面全体（E2E）にするほどではないが、Unit/Component では担保しにくい

## どこに置くか
- 同一 slice 内に co-locate（例: `features/x/ui/*.test.tsx`）
- テストのための cross-layer import はしない（Public API を守る）

命名規約（推奨）：
- `*.test.ts` / `*.test.tsx`
- 1ファイル=1対象（巨大化したら分割）

---

## E2E（Playwright）の範囲

原則：E2E は最小。

- 対象：主要導線（ログイン→主要ページ閲覧→重要操作）
- 目的：環境差分/統合不具合の検知（UI細部のスナップショット大会にしない）

---

## 新規feature追加時の“最低限テスト”

- `features/<slice>`：Component 1本（成功/失敗のどちらか）
- 重要な mutation がある：Integration 1本（MSWでスタブ）
- 主要導線に影響：E2E 1本（Playwright）

---

## 生成コード（Orval）に関する検査

目的：API 契約ズレや “手編集” の混入を防ぐ。

- `src/shared/api/generated/` は手編集禁止（差分が必要なら生成設定/上流 OpenAPI を直す）
- テストというより **CI の検査** として、以下のいずれかを導入する（プロジェクトで選択）：
	- `npm run api:generate` を実行し、生成物に差分が出ないことを確認（diff チェック）
	- OpenAPI を固定（リポジトリに同梱）し、生成を再現可能にする

---

## アーキテクチャ境界（FSD）を壊さない

目的：テストコードが “境界破壊の抜け道” にならないようにする。

- テストも Public API から import する（deep import で構造に依存しない）
- レイヤー/スライス境界の検査は ESLint（`eslint-plugin-boundaries`）を CI で必ず実行する

運用：
- 境界違反が出たら「直すべきは import の構造」であり、例外設定を安易に増やさない
