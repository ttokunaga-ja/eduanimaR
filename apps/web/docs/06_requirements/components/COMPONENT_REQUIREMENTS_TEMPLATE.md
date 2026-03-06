# Component Requirements Template

## Meta
- Component ID:
- Component Name:
- Intended FSD placement: shared/ui | entities/<slice>/ui | features/<slice>/ui | widgets/<slice>/ui
- Public API exposure: Yes/No（`index.ts` で公開するか）
- Status: draft / approved
- Last updated:

---

## 1) Purpose
- 何を表現/解決するコンポーネントか
- 想定利用箇所（どのページ/どの導線）

---

## 2) Non-goals
- やらないこと（肥大化防止）

---

## 3) Variants / States（Must）
- Loading
- Empty
- Error
- Disabled / Readonly
- Size / density / responsive

---

## 4) Interaction
- クリック/タップ
- キーボード操作（Tab/Enter/Escape/矢印など）
- フォーカス遷移

---

## 5) Content / i18n（Must）
- 表示文言の翻訳キー
- 文言の最大長・改行ポリシー（必要なら）

関連：`../../03_integration/I18N_LOCALE.md`

---

## 6) Accessibility（Must）
- 見出し/ランドマーク
- aria 属性
- 入力要素のラベル付け

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 7) Props Contract（High-level）
- Required props:
- Optional props:
- Events/callbacks:

注意：型の詳細はコード側で確定し、ここは「契約の意図」を書く。

---

## 8) Data Dependency
- API/Query に依存するか：Yes/No
- 依存する場合：どの層（DAL/feature）から注入されるべきか

関連：`../../01_architecture/DATA_ACCESS_LAYER.md`

---

## 9) Error Handling
- どのエラーを受け取り、どう表示するか
- エラーコード→表示の参照先

関連：
- `../../03_integration/ERROR_HANDLING.md`
- `../../03_integration/ERROR_CODES.md`

---

## 10) Testing Notes
- Unit（Vitest）で保証すること
- E2E（Playwright）で触るべき導線があるか

---

## 11) Acceptance Criteria（Must）
- [ ]
- [ ]
- [ ]

---

## 12) Open Questions
- 未確定事項
