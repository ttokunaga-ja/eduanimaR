# Component Requirements

## Meta
- Component ID: C_3
- Component Name: EvidenceSnippetCard
- Intended FSD placement: features/chat/ui
- Public API exposure: Yes（`index.ts` で公開）
- Status: approved
- Last updated: 2026-02-15
- **Updated**: 2026-02-14 - Backend DB Schema（evidence_snippets）と整合

---

## 1) Purpose
- AI回答の根拠（Evidence Snippet）を表示するクリック可能なカード
- ファイル名、引用テキスト（snippet）、ページ範囲、関連度スコアを表示
- クリックすると該当ファイルのプレビューまたはダウンロードを開始

**重要**: Backend DBでは `chats.evidence_snippets` (JSONB配列) として保存されている

想定利用箇所：
- ChatMessageList コンポーネント内（AI回答の下部）
- EvidenceList コンポーネント（一覧表示）

---

## 2) Non-goals
- ファイルのプレビュー機能そのもの（別コンポーネント）
- ファイルの編集・削除
- Evidence Snippet の評価・フィードバック（Phase 1 では不要）
- Material（チャンク）の直接表示（RawFileレベルで表示）

---

## 3) Variants / States（Must）

### 3.1 Default
- 通常表示（クリック可能）
- ホバー時に背景色を変更

### 3.2 Disabled
- Evidence Snippet が無効な場合（ファイルが削除された等）
- グレーアウト表示、クリック不可

### 3.3 Selected
- 最終回答に使用された Evidence Snippet を強調表示
- `selected_for_answer: true` の場合にハイライト

### 3.4 Size
- `compact`: リスト表示用（高さ 48px）
- `default`: カード表示用（高さ 80px）

### 3.5 Responsive
- Mobile: 全幅表示
- Desktop: カード幅固定（max-width: 400px）

---

## 4) Interaction

### 4.1 クリック
- Evidence Snippet カードをクリック → `onClick` コールバックを発火
- ファイルプレビューまたはダウンロードを開始（親コンポーネントで処理）

### 4.2 キーボード操作
- Tab: フォーカス移動
- Enter / Space: クリックと同じ動作

### 4.3 ホバー
- マウスホバー時に背景色を変更（視覚的フィードバック）
- 関連度スコアをツールチップで表示

---

## 5) Content / i18n（Must）

翻訳キー：
```json
{
  "features": {
    "evidenceSnippetCard": {
      "pageRange": "ページ {pageStart}",
      "pageRangeMulti": "ページ {pageStart}-{pageEnd}",
      "relevanceScore": "関連度: {score}%",
      "unavailable": "ファイルが利用できません",
      "selectedForAnswer": "回答に使用",
      "searchStep": "検索ステップ {step}"
    }
  }
}
```

関連：`../../02_tech_stack/I18N.md`

---

## 6) Accessibility（Must）

### 6.1 ボタン/リンク
- `<button role="button" aria-label="根拠: {snippet（先頭50文字）}, ページ {pageStart}-{pageEnd}, 関連度 {relevanceScore}">`

### 6.2 アイコン
- ファイルタイプアイコンに `aria-hidden="true"`（装飾的）
- スクリーンリーダーにはsnippetとページ範囲で通知

### 6.3 キーボードナビゲーション
- すべての Evidence Snippet カードがキーボードでアクセス可能
- フォーカス時に明確な視覚的インジケータを表示

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 7) Props Contract（High-level）

### Required props:
```typescript
{
  evidenceSnippet: EvidenceSnippet; // Evidence Snippet オブジェクト
  onClick: (evidenceSnippet: EvidenceSnippet) => void; // クリック時のコールバック
}
```

### Optional props:
```typescript
{
  size?: 'compact' | 'default'; // サイズバリアント（デフォルト: 'default'）
  disabled?: boolean;           // 無効化フラグ
  showRelevanceScore?: boolean; // 関連度スコアを表示するか（デフォルト: false）
  showSearchStep?: boolean;     // 検索ステップを表示するか（デフォルト: false）
  highlightSelected?: boolean;  // selected_for_answer をハイライトするか（デフォルト: true）
}
```

### Events/callbacks:
- `onClick`: Evidence Snippet カードをクリックした際に発火

---

## 8) Data Dependency

### API/Query に依存するか：No

Evidence Snippet オブジェクトは親コンポーネント（ChatMessageList）から Props として渡される。
このコンポーネント自体は API を呼び出さない。

データモデル（Backend DB Schema JSONB構造）：
```typescript
// EvidenceSnippet（chats.evidence_snippets の要素）
interface EvidenceSnippet {
  material_id: string;        // materialsテーブルのUUID
  snippet: string;            // 抽出されたテキストスニペット
  task_id: string;            // Phase 2で定義された調査タスクID
  relevance_score: number;    // 関連度スコア（0.0〜1.0）
  page_start?: number;        // 元ドキュメントのページ範囲（開始）
  page_end?: number;          // 元ドキュメントのページ範囲（終了）
  search_step: number;        // どの検索ステップで取得されたか（1〜5）
  selected_for_answer: boolean; // 最終回答生成に使用されたか
}
```

---

## 9) Error Handling

### どのエラーを受け取るか：
- `disabled` フラグで無効状態を表現
- ファイルが削除された場合は `disabled=true` で渡される

### 表示方針：
- 無効な Evidence Snippet はグレーアウトし、「ファイルが利用できません」と表示
- クリック不可

関連：
- `../../03_integration/ERROR_HANDLING.md`

---

## 10) Testing Notes

### Unit（Vitest）で保証すること：
- snippet（先頭100文字）の表示
- ページ範囲の表示（単一/範囲）
- クリック時のコールバック実行
- disabled 状態の表示
- selected_for_answer のハイライト表示
- 関連度スコアの表示（`showRelevanceScore=true` の場合）

### E2E（Playwright）で触るべき導線：
- Evidence Snippet カードクリック → ファイルプレビュー表示

---

## 11) Acceptance Criteria（Must）

- [ ] Evidence Snippet カードに snippet（先頭100文字）が表示される
- [ ] ページ範囲が表示される（単一: "ページ 15", 範囲: "ページ 15-17"）
- [ ] クリック可能で、onClick コールバックが発火する
- [ ] ホバー時に背景色が変わる
- [ ] disabled 状態でグレーアウト表示され、クリック不可になる
- [ ] `selected_for_answer: true` の場合にハイライト表示される
- [ ] `showRelevanceScore=true` で関連度スコアが表示される
- [ ] `showSearchStep=true` で検索ステップが表示される
- [ ] キーボードでフォーカス・操作できる
- [ ] スクリーンリーダーで「根拠: {snippet}, ページ {page}, 関連度 {score}」と読み上げられる
- [ ] `compact` サイズと `default` サイズが正しく表示される
- [ ] Backend DB Schema（evidence_snippets JSONB構造）と整合している

---

## 12) Open Questions

- ファイルプレビューは Modal か別ページか？
  - 回答: Modal で表示（Phase 1）
- 関連度スコアは常に表示するか？
  - 回答: デフォルトでは非表示、`showRelevanceScore` で切替可能
- material_id から raw_file_id への変換は誰が行うか？
  - 回答: 親コンポーネントまたはAPIレイヤーで実施（このコンポーネントは受け取るのみ）
