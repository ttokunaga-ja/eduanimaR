# Component Requirements

## Meta
- Component ID: C_4
- Component Name: FileTreeExplorer
- Intended FSD placement: features/fileManagement/ui
- Public API exposure: Yes（`index.ts` で公開）
- Status: approved
- Last updated: 2026-02-15
- **Updated**: 2026-02-14 - Backend DB Schema（raw_files）と整合

---

## 1) Purpose
- 科目ごとに保存済み資料をリスト形式で表示する（Phase 1 ではツリー形式不要）
- ファイルをクリックして開く、アップロード機能を提供
- 検索/フィルタ機能（ファイル名、ステータス）
- 解析状態（uploading / uploaded / processing / completed / failed / archived）を視覚的に表示

**重要**: Backend DBでは `raw_files` テーブルを使用（原本ファイル）

想定利用箇所：
- Chat Workspace の Sidebar（資料タブ）
- File Management Center ページ

---

## 2) Non-goals
- ファイルの編集機能（Phase 1 では不要）
- フォルダ構造の作成・管理（Phase 1 ではフラットなリスト）
- ファイルのバージョン管理
- 複数ファイルの一括操作（Phase 1 では個別操作のみ）

---

## 3) Variants / States（Must）

### 3.1 Loading
- 初回ローディング時は Skeleton UI を表示
- ファイル件数が不明なため、5件分のSkeleton を表示

### 3.2 Empty
- 科目内にファイルがない場合
- 「まだ資料がありません。アップロードしてください。」を表示
- アップロードボタンを強調

### 3.3 Filtered（検索/フィルタ適用中）
- 検索キーワードに一致するファイルのみ表示
- 「{count}件の資料が見つかりました」を表示
- 結果が0件の場合は「該当する資料が見つかりません」を表示

### 3.4 Error
- ファイル一覧取得失敗時
- 「資料の読み込みに失敗しました。再読み込みしてください。」を表示
- リトライボタンを提供

### 3.5 Responsive
- Desktop: 固定幅（サイドバー幅に合わせる）
- Mobile: 全幅表示

---

## 4) Interaction

### 4.1 ファイルクリック
- ファイルをクリック → プレビュー/ダウンロード（親コンポーネントで処理）

### 4.2 アップロード
- 「アップロード」ボタンをクリック → ファイル選択ダイアログを表示
- ドラッグ&ドロップにも対応（Phase 1 ではボタンのみ）

### 4.3 検索/フィルタ
- 検索ボックスに入力 → リアルタイムでフィルタリング
- フィルタ: 「すべて」「解析完了」「解析中」「失敗」

### 4.4 キーボード操作
- Tab: 次のファイルへフォーカス移動
- Enter: フォーカス中のファイルを開く
- 検索ボックス: 通常のテキスト入力

---

## 5) Content / i18n（Must）

翻訳キー：
```json
{
  "features": {
    "fileTreeExplorer": {
      "empty": "まだ資料がありません。アップロードしてください。",
      "uploadButton": "アップロード",
      "searchPlaceholder": "ファイル名で検索",
      "filterAll": "すべて",
      "filterCompleted": "解析完了",
      "filterProcessing": "解析中",
      "filterFailed": "失敗",
      "resultsCount": "{count}件の資料が見つかりました",
      "noResults": "該当する資料が見つかりません",
      "loadingError": "資料の読み込みに失敗しました。",
      "retryButton": "再読み込み",
      "statusPending": "待機中",
      "statusProcessing": "解析中",
      "statusCompleted": "完了",
      "statusFailed": "失敗"
    }
  }
}
```

関連：`../../02_tech_stack/I18N.md`

---

## 6) Accessibility（Must）

### 6.1 見出し/ランドマーク
- `<section aria-label="資料一覧">`
- 検索ボックス: `<input aria-label="ファイル名で検索">`

### 6.2 aria 属性
- 各ファイル: `<button role="button" aria-label="{filename}, {status}">`
- フィルタボタン: `aria-pressed="true"` で選択状態を表現

### 6.3 キーボードナビゲーション
- すべてのファイルがキーボードでアクセス可能
- 検索ボックスに Escape でクリア機能

関連：`../../01_architecture/ACCESSIBILITY.md`

---

## 7) Props Contract（High-level）

### Required props:
```typescript
{
  subjectId: string;          // 科目ID
  rawFiles: RawFile[];        // ファイル一覧
  onFileClick: (file: RawFile) => void; // ファイルクリック時のコールバック
  onUpload: () => void;       // アップロードボタンクリック時のコールバック
}
```

### Optional props:
```typescript
{
  isLoading?: boolean;        // ローディング状態
  error?: Error;              // エラー状態
  onRetry?: () => void;       // リトライコールバック
  searchable?: boolean;       // 検索機能を有効にするか（デフォルト: true）
  filterable?: boolean;       // フィルタ機能を有効にするか（デフォルト: true）
}
```

### Events/callbacks:
- `onFileClick`: ファイルをクリックした際に発火
- `onUpload`: アップロードボタンをクリックした際に発火

---

## 8) Data Dependency

### API/Query に依存するか：Yes

**依存する層**：
- `features/fileManagement/lib/useRawFiles` Hook から `rawFiles` を注入
- `features/fileManagement/lib/useUploadFile` Hook から `onUpload` を注入

データフロー：
1. `useRawFiles(subjectId)` でファイル一覧を取得
2. Props として渡す
3. ポーリング（5秒間隔）で解析状態を更新（TanStack Query の `refetchInterval`）

データモデル（Backend DB Schemaと一致）：
```typescript
// RawFile（原本ファイル）
interface RawFile {
  id: string;                // UUID (内部ID)
  nanoid: string;            // 20文字の外部公開ID
  user_id: string;           // 所有者UUID（物理制約）
  subject_id: string;        // 所属科目UUID（物理制約）
  original_filename: string; // 元のファイル名
  file_type: FileType;       // ファイルタイプ（ENUM）
  file_size_bytes: number;   // ファイルサイズ（バイト）
  status: FileStatus;        // ファイルステータス
  total_pages?: number;      // PDF/PowerPointのページ数
  processed_at?: string;     // 処理完了日時（ISO 8601）
  created_at: string;        // ISO 8601
  is_active: boolean;        // ソフトデリート用
}

// FileStatus（Backend ENUM型と一致）
type FileStatus =
  | 'uploading'      // アップロード中
  | 'uploaded'       // アップロード完了
  | 'processing'     // 処理中（Vision Reasoning実行中）
  | 'completed'      // 処理完了
  | 'failed'         // 処理失敗
  | 'archived';      // アーカイブ済み

// FileType（Backend ENUM型と一致、抜粋）
type FileType =
  | 'pdf' | 'text'
  | 'python' | 'go' | 'javascript'
  | 'png' | 'jpeg' | 'webp'
  | 'docx' | 'xlsx' | 'pptx'
  | 'google_docs' | 'google_sheets' | 'google_slides'
  | 'other';
```

関連：`../../01_architecture/DATA_MODELS.md`

---

## 9) Error Handling

### どのエラーを受け取るか：
- `useRawFiles` の fetch エラー → `error` Prop
- アップロード失敗 → `useUploadFile` 内で処理（このコンポーネントには影響しない）

### 表示方針：
- ファイル一覧取得失敗: 全体にエラー表示 + リトライボタン
- 個別ファイルの解析失敗: `status="failed"` で視覚的に表示

関連：
- `../../03_integration/ERROR_HANDLING.md`
- `../../03_integration/ERROR_CODES.md`

---

## 10) Testing Notes

### Unit（Vitest）で保証すること：
- Empty 状態の表示
- Loading 状態（Skeleton）の表示
- 検索フィルタリング
- ステータスフィルタリング

### Integration（Vitest + MSW）で保証すること：
- `useRawFiles` との結合
- ファイルクリック時のコールバック実行

### E2E（Playwright）で触るべき導線：
- ファイルアップロード → 解析完了 → ファイルクリック → プレビュー

---

## 11) Acceptance Criteria（Must）

- [ ] 科目内のファイル一覧がリスト形式で表示される（Phase 1 ではツリー不要）
- [ ] 各ファイルにファイル名、タイプアイコン、解析状態（uploading/uploaded/processing/completed/failed/archived）が表示される
- [ ] ファイルをクリックすると `onFileClick` コールバックが発火する
- [ ] アップロードボタンをクリックすると `onUpload` コールバックが発火する
- [ ] 検索ボックスでファイル名をリアルタイム検索できる
- [ ] ステータスフィルタ（すべて/完了/処理中/失敗/アーカイブ済み）が機能する
- [ ] Empty 状態で適切なメッセージとアップロードボタンが表示される
- [ ] Loading 状態で Skeleton UI が表示される
- [ ] エラー時に適切なメッセージとリトライボタンが表示される
- [ ] 解析状態がポーリングで更新される
- [ ] キーボードでファイルをナビゲーションできる
- [ ] Backend DB Schema（raw_files テーブル）と整合している

---

## 12) Open Questions

- ツリー表示は Phase 1 で実装するか？
  - 回答: **Phase 1 ではフラットなリスト**。Phase 2 でフォルダ構造を検討
- ポーリング間隔は？
  - 回答: 5秒間隔（TanStack Query の `refetchInterval: 5000`）
- FileStatus ENUM の全種類に対応するか？
  - 回答: Yes。Backend ENUM型（uploading/uploaded/processing/completed/failed/archived）すべてに対応
