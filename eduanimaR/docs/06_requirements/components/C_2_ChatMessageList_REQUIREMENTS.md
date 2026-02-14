# Component Requirements

## Meta
- Component ID: C_2
- Component Name: ChatMessageList
- Intended FSD placement: features/chat/ui
- Public API exposure: Yes（`index.ts` で公開）
- Status: approved
- Last updated: 2026-02-15
- **Updated**: 2026-02-14 - Backend DB Schema と整合

---

## 1) Purpose
- 科目内のチャット（質問と回答）を時系列順に表示する
- ユーザーの質問とAI回答を視覚的に区別する
- AI回答に含まれる根拠（Evidence Snippet）を表示し、クリック可能にする
- SSEストリーミング中の部分的な回答表示に対応する

**重要**: Backend DBでは「Thread/Message」ではなく「Chat」モデルを使用（1つのChatに1つの質問と1つの回答）

想定利用箇所：
- Chat Workspace ページ（メインコンテンツエリア）
- Chrome拡張の Sidepanel（Phase 2）

---

## 2) Non-goals
- チャットの編集・削除機能（Phase 1 では不要）
- チャットのリアクション（いいね、引用等）
- 科目横断の全文検索（それは History Archive で行う）
- 複数ユーザー間のチャット（個人利用のみ）
- **会話形式のスレッド**（1質問1回答のみ）

---

## 3) Variants / States（Must）

### 3.1 Loading
- 初期ローディング時は Skeleton UI を表示
- チャット件数が不明なため、3〜5件分のSkeleton を表示

### 3.2 Empty
- 科目内にチャットがない場合
- 「まだ質問がありません。質問を入力してください。」を表示

### 3.3 Streaming（SSE中）
- AI回答がストリーミング中の場合
- 部分的な `final_answer_markdown` を逐次追加表示
- StreamingIndicator（C_5）を表示
- Evidence Snippets は確定後（done イベント）に表示

### 3.4 Error
- チャット取得失敗時
- 「チャットの読み込みに失敗しました。再読み込みしてください。」を表示
- リトライボタンを提供

### 3.5 Responsive
- Desktop: 左右に余白、中央寄せ（max-width: 800px）
- Mobile: 全幅表示、左右パディング 16px

---

## 4) Interaction

### 4.1 スクロール
- 新しいチャットが追加されたら、自動で最下部へスクロール
- ストリーミング中も最下部を維持
- ユーザーが手動スクロールした場合は自動スクロールを停止

### 4.2 Evidence Snippet のクリック
- AI回答内の Evidence Snippet カード（C_3）をクリック
- 該当ファイルのプレビュー/ダウンロードを表示（別コンポーネント）

### 4.3 キーボード操作
- Tab: 次の Evidence Snippet へフォーカス移動
- Enter / Space: フォーカス中の Evidence Snippet を開く

---

## 5) Content / i18n（Must）

翻訳キー：
```json
{
  "features": {
    "chatMessageList": {
      "empty": "まだ質問がありません。質問を入力してください。",
      "loadingError": "チャットの読み込みに失敗しました。",
      "retryButton": "再読み込み",
      "questionLabel": "質問",
      "answerLabel": "回答",
      "streamingLabel": "生成中...",
      "evidenceLabel": "根拠"
    }
  }
}
```

関連：`../../02_tech_stack/I18N.md`

---

## 6) Accessibility（Must）

### 6.1 見出し/ランドマーク
- `<section role="log" aria-live="polite" aria-label="チャット一覧">`
  - ストリーミング中の更新を支援技術に通知

### 6.2 aria 属性
- 各チャット: `<article role="article">`
- 質問: `aria-label="質問: {question}"`
- 回答: `aria-label="回答"`
- Evidence Snippet リンク: `aria-label="根拠: {filename}, ページ {page_start}-{page_end}"`

### 6.3 キーボードナビゲーション
- Evidence Snippet カードはすべてキーボードでアクセス可能
- Tab 順序: 上から下、左から右

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 7) Props Contract（High-level）

### Required props:
```typescript
{
  subjectId: string;         // 科目ID
  chats: Chat[];             // 表示するチャット一覧
  streamingChat?: StreamingChat; // ストリーミング中のチャット（任意）
}
```

### Optional props:
```typescript
{
  onEvidenceClick?: (evidence: EvidenceSnippet) => void; // Evidence クリック時のコールバック
  isLoading?: boolean;        // ローディング状態
  error?: Error;              // エラー状態
  onRetry?: () => void;       // リトライコールバック
}
```

### Events/callbacks:
- `onEvidenceClick`: Evidence Snippet カードをクリックした際に発火

注意：型の詳細はコード側で確定し、ここは「契約の意図」を書く。

---

## 8) Data Dependency

### API/Query に依存するか：Yes

**依存する層**：
- `features/chat/lib/useChats` Hook から `chats` を注入
- `features/chat/lib/useSendQuestion` Hook から `streamingChat` を注入

データフロー：
1. `useChats(subjectId)` で既存チャットを取得
2. `useSendQuestion(subjectId)` でストリーミング状態を取得
3. 両方を Props として渡す

データモデル（Backend DB Schemaと一致）：
```typescript
// Chat（1質問1回答）
interface Chat {
  id: string;                // UUID
  nanoid: string;            // 20文字の外部公開ID
  user_id: string;           // 質問者UUID
  subject_id: string;        // 科目UUID
  question: string;          // 質問内容
  final_answer_markdown?: string; // 回答（Markdown）
  evidence_snippets?: EvidenceSnippet[]; // 根拠スニペット
  used_raw_file_ids: string[]; // 使用ファイルUUID配列
  created_at: string;        // ISO 8601
  completed_at?: string;     // 完了日時
}

// Evidence Snippet（Backend JSONB構造）
interface EvidenceSnippet {
  material_id: string;        // materialsテーブルのUUID
  snippet: string;            // 抽出されたテキスト
  task_id: string;            // 調査タスクID
  relevance_score: number;    // 関連度（0.0〜1.0）
  page_start?: number;        // ページ範囲（開始）
  page_end?: number;          // ページ範囲（終了）
  search_step: number;        // 検索ステップ（1〜5）
  selected_for_answer: boolean; // 最終回答に使用されたか
}
```

関連：`../../01_architecture/DATA_MODELS.md`

---

## 9) Error Handling

### どのエラーを受け取るか：
- `useChats` の fetch エラー → `error` Prop
- ストリーミング接続エラー → `streamingChat.error`

### 表示方針：
- チャット取得失敗: 全体にエラー表示 + リトライボタン
- ストリーミングエラー: 該当チャットに「生成に失敗しました」を表示

関連：
- `../../03_integration/ERROR_HANDLING.md`
- `../../03_integration/ERROR_CODES.md`

---

## 10) Testing Notes

### Unit（Vitest）で保証すること：
- Empty 状態の表示
- Loading 状態（Skeleton）の表示
- チャットの時系列ソート
- 質問/回答の視覚的区別

### Integration（Vitest + MSW）で保証すること：
- `useChats` との結合
- ストリーミング状態の表示
- Evidence Snippet クリック時のコールバック実行

### E2E（Playwright）で触るべき導線：
- 質問送信 → ストリーミング表示 → 完了
- Evidence Snippet クリック → プレビュー表示

---

## 11) Acceptance Criteria（Must）

- [ ] 科目内のチャットが時系列順に表示される
- [ ] ユーザーの質問とAI回答が視覚的に区別される（アバター、背景色等）
- [ ] AI回答に Evidence Snippets が表示される（SourceCitation コンポーネント使用）
- [ ] SSEストリーミング中、部分的な回答が逐次追加表示される
- [ ] ストリーミング完了後、Evidence Snippets が表示される
- [ ] Evidence Snippet カードがクリック可能で、コールバックが発火する
- [ ] Empty 状態で適切なメッセージが表示される
- [ ] Loading 状態で Skeleton UI が表示される
- [ ] エラー時に適切なメッセージとリトライボタンが表示される
- [ ] 新しいチャット追加時に自動で最下部へスクロールする
- [ ] キーボードで Evidence Snippets をナビゲーションできる
- [ ] スクリーンリーダーで各チャットの内容が読み上げられる
- [ ] Backend DB Schema（Chat モデル）と整合している

---

## 12) Open Questions

- チャットの最大表示件数は？（無制限 or ページネーション）
  - 回答: Phase 1 では無制限。Phase 2 で仮想スクロールを検討
- Evidence Snippet のプレビュー表示は別コンポーネントか？
  - 回答: Yes。`features/filePreview/ui/FilePreview` を使用
- 1チャット=1質問1回答で確定か？
  - 回答: Yes。Backend DB Schemaに従う（会話形式のスレッドは不要）
