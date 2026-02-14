# Component Requirements

## Meta
- Component ID: C_2
- Component Name: ChatMessageList
- Intended FSD placement: features/chat/ui
- Public API exposure: Yes（`index.ts` で公開）
- Status: approved
- Last updated: 2026-02-15

---

## 1) Purpose
- Q&Aスレッド内のメッセージ一覧を時系列順に表示する
- ユーザーメッセージとAI回答を視覚的に区別する
- AI回答に含まれる Source（根拠）を表示し、クリック可能にする
- SSEストリーミング中の部分的なメッセージ表示に対応する

想定利用箇所：
- Chat Workspace ページ（メインコンテンツエリア）
- Chrome拡張の Sidepanel（Phase 2）

---

## 2) Non-goals
- メッセージの編集・削除機能（Phase 1 では不要）
- メッセージのリアクション（いいね、引用等）
- スレッド横断の全文検索（それは History Archive で行う）
- 複数ユーザー間のチャット（個人利用のみ）

---

## 3) Variants / States（Must）

### 3.1 Loading
- 初期ローディング時は Skeleton UI を表示
- メッセージ件数が不明なため、3〜5件分のSkeleton を表示

### 3.2 Empty
- スレッド内にメッセージがない場合
- 「まだメッセージがありません。質問を入力してください。」を表示

### 3.3 Streaming（SSE中）
- AI回答がストリーミング中の場合
- 部分的な content を逐次追加表示
- StreamingIndicator（C_5）を表示
- Source は確定後（done イベント）に表示

### 3.4 Error
- メッセージ取得失敗時
- 「メッセージの読み込みに失敗しました。再読み込みしてください。」を表示
- リトライボタンを提供

### 3.5 Responsive
- Desktop: 左右に余白、中央寄せ（max-width: 800px）
- Mobile: 全幅表示、左右パディング 16px

---

## 4) Interaction

### 4.1 スクロール
- 新しいメッセージが追加されたら、自動で最下部へスクロール
- ストリーミング中も最下部を維持
- ユーザーが手動スクロールした場合は自動スクロールを停止

### 4.2 Source のクリック
- AI回答内の Source カード（C_3）をクリック
- 該当ファイルのプレビュー/ダウンロードを表示（別コンポーネント）

### 4.3 キーボード操作
- Tab: 次の Source へフォーカス移動
- Enter / Space: フォーカス中の Source を開く

---

## 5) Content / i18n（Must）

翻訳キー：
```json
{
  "features": {
    "chatMessageList": {
      "empty": "まだメッセージがありません。質問を入力してください。",
      "loadingError": "メッセージの読み込みに失敗しました。",
      "retryButton": "再読み込み",
      "userLabel": "あなた",
      "aiLabel": "AI",
      "streamingLabel": "生成中..."
    }
  }
}
```

関連：`../../02_tech_stack/I18N.md`

---

## 6) Accessibility（Must）

### 6.1 見出し/ランドマーク
- `<section role="log" aria-live="polite" aria-label="チャットメッセージ">`
  - ストリーミング中の更新を支援技術に通知

### 6.2 aria 属性
- 各メッセージ: `<article role="article">`
- ユーザー/AI の区別: `aria-label="あなたのメッセージ"` / `aria-label="AIの回答"`
- Source リンク: `aria-label="根拠: {filename}, ページ {page_number}"`

### 6.3 キーボードナビゲーション
- Source カードはすべてキーボードでアクセス可能
- Tab 順序: 上から下、左から右

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 7) Props Contract（High-level）

### Required props:
```typescript
{
  threadId: string;           // スレッドID
  messages: Message[];        // 表示するメッセージ一覧
  streamingMessage?: StreamingMessage; // ストリーミング中のメッセージ（任意）
}
```

### Optional props:
```typescript
{
  onSourceClick?: (source: Source) => void; // Source クリック時のコールバック
  isLoading?: boolean;        // ローディング状態
  error?: Error;              // エラー状態
  onRetry?: () => void;       // リトライコールバック
}
```

### Events/callbacks:
- `onSourceClick`: Source カードをクリックした際に発火

注意：型の詳細はコード側で確定し、ここは「契約の意図」を書く。

---

## 8) Data Dependency

### API/Query に依存するか：Yes

**依存する層**：
- `features/chat/lib/useMessages` Hook から `messages` を注入
- `features/chat/lib/useSendQuestion` Hook から `streamingMessage` を注入

データフロー：
1. `useMessages(threadId)` で既存メッセージを取得
2. `useSendQuestion(threadId)` でストリーミング状態を取得
3. 両方を Props として渡す

関連：`../../01_architecture/DATA_ACCESS_LAYER.md`

---

## 9) Error Handling

### どのエラーを受け取るか：
- `useMessages` の fetch エラー → `error` Prop
- ストリーミング接続エラー → `streamingMessage.error`

### 表示方針：
- メッセージ取得失敗: 全体にエラー表示 + リトライボタン
- ストリーミングエラー: 該当メッセージに「生成に失敗しました」を表示

関連：
- `../../03_integration/ERROR_HANDLING.md`
- `../../03_integration/ERROR_CODES.md`

---

## 10) Testing Notes

### Unit（Vitest）で保証すること：
- Empty 状態の表示
- Loading 状態（Skeleton）の表示
- メッセージの時系列ソート
- ユーザー/AI の視覚的区別

### Integration（Vitest + MSW）で保証すること：
- `useMessages` との結合
- ストリーミング状態の表示
- Source クリック時のコールバック実行

### E2E（Playwright）で触るべき導線：
- 質問送信 → ストリーミング表示 → 完了
- Source クリック → プレビュー表示

---

## 11) Acceptance Criteria（Must）

- [ ] スレッド内のメッセージが時系列順に表示される
- [ ] ユーザーメッセージとAI回答が視覚的に区別される（アバター、背景色等）
- [ ] AI回答に Source が表示される（SourceCitation コンポーネント使用）
- [ ] SSEストリーミング中、部分的なコンテンツが逐次追加表示される
- [ ] ストリーミング完了後、Source が表示される
- [ ] Source カードがクリック可能で、コールバックが発火する
- [ ] Empty 状態で適切なメッセージが表示される
- [ ] Loading 状態で Skeleton UI が表示される
- [ ] エラー時に適切なメッセージとリトライボタンが表示される
- [ ] 新しいメッセージ追加時に自動で最下部へスクロールする
- [ ] キーボードで Source をナビゲーションできる
- [ ] スクリーンリーダーで各メッセージの内容が読み上げられる

---

## 12) Open Questions

- メッセージの最大表示件数は？（無制限 or ページネーション）
  - 回答: Phase 1 では無制限。Phase 2 で仮想スクロールを検討
- Source のプレビュー表示は別コンポーネントか？
  - 回答: Yes。`features/filePreview/ui/FilePreview` を使用
