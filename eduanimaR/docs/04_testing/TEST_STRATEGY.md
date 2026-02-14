# Test Strategy

このドキュメントは、テスト範囲と責務を固定し、AI が過剰な E2E や不適切な層にテストを書かないようにするためのガイドです。

関連：
- FSD 境界：`../skills/SKILL_FSD_BOUNDARIES.md`
- SSE クライアント：`../skills/SKILL_SSE_CLIENT.md`
- Vitest：`../skills/SKILL_TESTING_VITEST.md`
- Playwright：`../skills/SKILL_TESTING_PLAYWRIGHT.md`

---

## 結論（Must）

- Unit / Component / Integration: **Vitest**（カバレッジ **80%以上**）
- E2E: **Playwright**（主要導線のみ）
- API Mock: **MSW**（Mock Service Worker）
- SSE ストリーミングのテストは専用パターンで実装
- Chrome拡張のテスト（Phase 2 で追加）

---

## 採用ツール（固定）

### 1) Vitest
- Unit / Component / Integration テスト
- 高速な実行速度（Vite ベース）
- TypeScript ネイティブサポート

### 2) React Testing Library
- Component testing の標準
- \`@testing-library/react\` + \`@testing-library/user-event\`
- ユーザー視点のテストを推奨

### 3) MSW（Mock Service Worker）
- API スタブ/モック
- ブラウザとNode.jsの両方で動作
- Orval の MSW 生成機能を活用可能

### 4) Playwright
- E2E テスト
- クロスブラウザ対応（Chromium / Firefox / WebKit）
- ヘッドレス実行可能

---

## 目的

- 壊れやすい箇所（仕様・依存境界）を優先してテストする
- FSD の層構造を壊さない（テストが層違反を誘発しない）
- カバレッジ 80%以上を維持する

---

## テストの種類（最小）

### 1) Unit Test

**対象**: \`shared/lib\`, \`entities/*/lib\` など純粋関数

**観点**：
- 入力→出力の正しさ
- エッジケース（null / undefined / 空配列等）
- エラーハンドリング

**禁止**: DOM を伴う結合テストを Unit に押し込む

---

### 2) Component Test

**対象**: \`shared/ui\`, \`entities/*/ui\`, \`features/*/ui\`

**指針**: ユーザー操作と表示を中心に（実装詳細に寄せない）

**最低限の観点**：
- 主要な表示（ラベル/必須要素）
- 主要な操作（入力→送信→状態変化）
- エラー表示（バリデーション / API失敗）

---

### 3) Integration Test（必要な場合のみ）

**対象**: \`features\` と API Hook の結合

**方針**: API は MSW 等でスタブ（プロジェクト採用に合わせて追記）

**Integration を書く基準**：
- 重要なユースケースで「APIとUIの接続」が壊れると影響が大きい
- 画面全体（E2E）にするほどではないが、Unit/Component では担保しにくい

---

## SSE ストリーミングのテスト

### 方針

SSE は通常の HTTP リクエストと異なるため、専用のテストパターンが必要。

### 実装方針

- \`EventSource\` をモックして、ストリーミングイベントをシミュレート
- 部分的なコンテンツの受信と状態更新を検証
- 完了イベントでキャッシュ更新を検証

---

## Chrome拡張のテスト（Phase 2）

### 方針

Chrome拡張のテストは、Web版とは異なる環境で実行する必要がある。

### ツール

- **Vitest** + **@webext-core/fake-browser**（Chrome API のモック）
- **Playwright**（E2E / Content Script のテスト）

### 対象

1. **Background Service Worker**
   - API 通信のロジック
   - メッセージパッシング

2. **Content Script**
   - DOM 監視
   - LMS からの資料検知

3. **Sidepanel / Popup**
   - UI コンポーネント（React Testing Library）

---

## カバレッジ目標

### 目標値

- **Unit Test**: 90%以上
- **Component Test**: 80%以上
- **Integration Test**: 70%以上
- **全体**: 80%以上

### CI での検証

CI で \`npm run test:coverage\` を実行し、閾値をチェックする。

---

## どこに置くか

- 同一 slice 内に co-locate（例: \`features/x/ui/*.test.tsx\`）
- テストのための cross-layer import はしない（Public API を守る）

命名規約（推奨）：
- \`*.test.ts\` / \`*.test.tsx\`
- 1ファイル=1対象（巨大化したら分割）

---

## E2E（Playwright）の範囲

原則：E2E は最小。

**対象**：主要導線（ログイン→主要ページ閲覧→重要操作）

**目的**：環境差分/統合不具合の検知（UI細部のスナップショット大会にしない）

---

## 新規feature追加時の"最低限テスト"

- \`features/<slice>\`：Component 1本（成功/失敗のどちらか）
- 重要な mutation がある：Integration 1本（MSWでスタブ）
- 主要導線に影響：E2E 1本（Playwright）

---

## 生成コード（Orval）に関する検査

目的：API 契約ズレや "手編集" の混入を防ぐ。

- \`src/shared/api/generated/\` は手編集禁止（差分が必要なら生成設定/上流 OpenAPI を直す）
- テストというより **CI の検査** として、以下のいずれかを導入する（プロジェクトで選択）：
- \`npm run api:generate\` を実行し、生成物に差分が出ないことを確認（diff チェック）
- OpenAPI を固定（リポジトリに同梱）し、生成を再現可能にする

---

## アーキテクチャ境界（FSD）を壊さない

目的：テストコードが "境界破壊の抜け道" にならないようにする。

- テストも Public API から import する（deep import で構造に依存しない）
- レイヤー/スライス境界の検査は ESLint（\`eslint-plugin-boundaries\`）を CI で必ず実行する

運用：
- 境界違反が出たら「直すべきは import の構造」であり、例外設定を安易に増やさない

---

## 禁止（AI/人間共通）

- DOM を伴う結合テストを Unit に押し込む
- テストで FSD の境界を破る（deep import）
- E2E で UI 細部のスナップショットテストを大量に書く
- カバレッジを上げるためだけの無意味なテストを書く
- SSE ストリーミングのテストを省略する

---

## 実装チェックリスト

- [ ] Unit Test が 90%以上のカバレッジを達成しているか？
- [ ] Component Test がユーザー視点で書かれているか？
- [ ] Integration Test で API と UI の結合が検証されているか？
- [ ] SSE ストリーミングのテストが実装されているか？
- [ ] E2E Test が主要導線をカバーしているか？
- [ ] MSW で API がモックされているか？
- [ ] テストが FSD の Public API を守っているか？
- [ ] CI でカバレッジチェックが実行されているか？
