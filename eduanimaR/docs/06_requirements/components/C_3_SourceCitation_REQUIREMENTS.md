# Component Requirements

## Meta
- Component ID: C_3
- Component Name: SourceCitation
- Intended FSD placement: features/chat/ui
- Public API exposure: Yes（`index.ts` で公開）
- Status: approved
- Last updated: 2026-02-15

---

## 1) Purpose
- AI回答の根拠（Source）を表示するクリック可能なリンクカード
- ファイル名、引用箇所（excerpt）、アイコン（ファイルタイプ別）を表示
- クリックすると該当ファイルのプレビューまたはダウンロードを開始

想定利用箇所：
- ChatMessageList コンポーネント内（AI回答の下部）
- SourceList コンポーネント（一覧表示）

---

## 2) Non-goals
- ファイルのプレビュー機能そのもの（別コンポーネント）
- ファイルの編集・削除
- Source の評価・フィードバック（Phase 1 では不要）

---

## 3) Variants / States（Must）

### 3.1 Default
- 通常表示（クリック可能）
- ホバー時に背景色を変更

### 3.2 Disabled
- Source が無効な場合（ファイルが削除された等）
- グレーアウト表示、クリック不可

### 3.3 Size
- `compact`: リスト表示用（高さ 48px）
- `default`: カード表示用（高さ 80px）

### 3.4 Responsive
- Mobile: 全幅表示
- Desktop: カード幅固定（max-width: 300px）

---

## 4) Interaction

### 4.1 クリック
- Source カードをクリック → `onClick` コールバックを発火
- ファイルプレビューまたはダウンロードを開始（親コンポーネントで処理）

### 4.2 キーボード操作
- Tab: フォーカス移動
- Enter / Space: クリックと同じ動作

### 4.3 ホバー
- マウスホバー時に背景色を変更（視覚的フィードバック）

---

## 5) Content / i18n（Must）

翻訳キー：
```json
{
  "features": {
    "sourceCitation": {
      "pageLabel": "ページ {page}",
      "relevanceScore": "関連度: {score}%",
      "unavailable": "ファイルが利用できません"
    }
  }
}
```

関連：`../../02_tech_stack/I18N.md`

---

## 6) Accessibility（Must）

### 6.1 ボタン/リンク
- `<button role="button" aria-label="根拠: {filename}, ページ {page_number}">`
- または `<a role="link" aria-label="...">`（実装による）

### 6.2 アイコン
- ファイルタイプアイコンに `aria-hidden="true"`（装飾的）
- スクリーンリーダーにはファイル名で通知

### 6.3 キーボードナビゲーション
- すべての Source カードがキーボードでアクセス可能
- フォーカス時に明確な視覚的インジケータを表示

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 7) Props Contract（High-level）

### Required props:
```typescript
{
  source: Source;             // Source オブジェクト
  onClick: (source: Source) => void; // クリック時のコールバック
}
```

### Optional props:
```typescript
{
  size?: 'compact' | 'default'; // サイズバリアント（デフォルト: 'default'）
  disabled?: boolean;           // 無効化フラグ
  showRelevanceScore?: boolean; // 関連度スコアを表示するか（デフォルト: false）
}
```

### Events/callbacks:
- `onClick`: Source カードをクリックした際に発火

---

## 8) Data Dependency

### API/Query に依存するか：No

Source オブジェクトは親コンポーネント（ChatMessageList）から Props として渡される。
このコンポーネント自体は API を呼び出さない。

---

## 9) Error Handling

### どのエラーを受け取るか：
- `disabled` フラグで無効状態を表現
- ファイルが削除された場合は `disabled=true` で渡される

### 表示方針：
- 無効な Source はグレーアウトし、「ファイルが利用できません」と表示
- クリック不可

関連：
- `../../03_integration/ERROR_HANDLING.md`

---

## 10) Testing Notes

### Unit（Vitest）で保証すること：
- ファイルタイプ別のアイコン表示
- クリック時のコールバック実行
- disabled 状態の表示
- 関連度スコアの表示（`showRelevanceScore=true` の場合）

### E2E（Playwright）で触るべき導線：
- Source カードクリック → ファイルプレビュー表示

---

## 11) Acceptance Criteria（Must）

- [ ] Source カードにファイル名が表示される
- [ ] ファイルタイプに応じたアイコンが表示される（PDF/PPTX/DOCX/TXT）
- [ ] 引用箇所（excerpt）が表示される（最大100文字、省略記号付き）
- [ ] ページ番号が表示される（PDFの場合）
- [ ] クリック可能で、onClick コールバックが発火する
- [ ] ホバー時に背景色が変わる
- [ ] disabled 状態でグレーアウト表示され、クリック不可になる
- [ ] キーボードでフォーカス・操作できる
- [ ] スクリーンリーダーで「根拠: {filename}, ページ {page}」と読み上げられる
- [ ] `compact` サイズと `default` サイズが正しく表示される

---

## 12) Open Questions

- ファイルプレビューは Modal か別ページか？
  - 回答: Modal で表示（Phase 1）
- 関連度スコアは常に表示するか？
  - 回答: デフォルトでは非表示、`showRelevanceScore` で切替可能
