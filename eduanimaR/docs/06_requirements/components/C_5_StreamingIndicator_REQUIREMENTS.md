# Component Requirements

## Meta
- Component ID: C_5
- Component Name: StreamingIndicator
- Intended FSD placement: features/chat/ui
- Public API exposure: Yes（`index.ts` で公開）
- Status: approved
- Last updated: 2026-02-15

---

## 1) Purpose
- SSEストリーミング中にAI回答が生成されていることをユーザーに視覚的に伝える
- アニメーション付きのインジケータを表示し、待機状態であることを明示

想定利用箇所：
- ChatMessageList コンポーネント内（ストリーミング中のメッセージ）
- 質問入力欄の送信ボタン横（送信中の状態表示）

---

## 2) Non-goals
- プログレスバー（進捗パーセンテージ）の表示（ストリーミングは進捗不明）
- キャンセルボタン（Phase 1 では不要）
- エラー表示（親コンポーネントで処理）

---

## 3) Variants / States（Must）

### 3.1 Default（アニメーション）
- 3つのドット（●●●）が左から右へ順番に点滅するアニメーション
- または回転スピナー（デザインによる）

### 3.2 Size
- `small`: 高さ 16px（インライン表示用）
- `default`: 高さ 24px（単独表示用）

### 3.3 With Label
- インジケータの横にラベルを表示（例: "生成中..."）
- `showLabel` プロップで制御

### 3.4 Responsive
- すべてのサイズでレスポンシブ対応（自動調整）

---

## 4) Interaction

### 4.1 インタラクションなし
- このコンポーネントは表示のみで、ユーザー操作を受け付けない
- クリックやホバーのイベントは不要

---

## 5) Content / i18n（Must）

翻訳キー：
```json
{
  "features": {
    "streamingIndicator": {
      "label": "生成中...",
      "ariaLabel": "AI回答を生成中です"
    }
  }
}
```

関連：`../../02_tech_stack/I18N.md`

---

## 6) Accessibility（Must）

### 6.1 aria 属性
- `<div role="status" aria-live="polite" aria-label="AI回答を生成中です">`
- ストリーミング開始時にスクリーンリーダーへ通知

### 6.2 アニメーション
- `prefers-reduced-motion: reduce` を尊重し、アニメーションを無効化可能にする
- CSSで `@media (prefers-reduced-motion: reduce)` を使用

### 6.3 視覚的明確さ
- 背景とのコントラスト比を4.5:1以上に保つ

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 7) Props Contract（High-level）

### Required props:
なし（すべて Optional）

### Optional props:
```typescript
{
  size?: 'small' | 'default';   // サイズバリアント（デフォルト: 'default'）
  showLabel?: boolean;          // ラベルを表示するか（デフォルト: false）
  label?: string;               // カスタムラベル（デフォルト: 翻訳キーから取得）
  className?: string;           // 追加のCSSクラス
}
```

### Events/callbacks:
なし（表示のみ）

---

## 8) Data Dependency

### API/Query に依存するか：No

完全にプレゼンテーショナルなコンポーネント。
データは親コンポーネントから Props として渡される。

---

## 9) Error Handling

このコンポーネント自体はエラーを扱わない。
ストリーミングエラーは親コンポーネント（ChatMessageList）で処理される。

---

## 10) Testing Notes

### Unit（Vitest）で保証すること：
- サイズバリアント（small / default）の表示
- ラベルの表示/非表示
- `prefers-reduced-motion` 対応（アニメーション無効化）

### E2E（Playwright）で触るべき導線：
- 質問送信 → ストリーミングインジケータが表示される → 完了後に消える

---

## 11) Acceptance Criteria（Must）

- [ ] ストリーミング中にアニメーション付きインジケータが表示される
- [ ] `small` サイズと `default` サイズが正しく表示される
- [ ] `showLabel=true` でラベル「生成中...」が表示される
- [ ] `prefers-reduced-motion: reduce` 時にアニメーションが無効化される
- [ ] スクリーンリーダーで「AI回答を生成中です」と読み上げられる
- [ ] 背景とのコントラスト比が4.5:1以上

---

## 12) Open Questions

- アニメーションはドット点滅かスピナーか？
  - 回答: ドット点滅（●●●）を推奨。デザインチームと調整
- Phase 2 でキャンセルボタンを追加するか？
  - 回答: Phase 2 で検討
