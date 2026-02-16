# QA Page Requirements

Last-updated: 2026-02-16

## 目的
AI Agentによる質問対応の単一インターフェース。すべてのユースケース（資料収集、質問明確化、小テスト解説、直接回答）を同じUIで実現。

---

## ユースケース

### 1. 資料収集依頼
- **入力**: 「統計学の資料を集めて」
- **Agent動作**: 広範囲検索 → 資料リスト提示
- **UI**: 複数の資料カードを表示、クリックで詳細表示

### 2. 曖昧な質問
- **入力**: 「決定係数って何？」
- **Agent動作**: 複数候補検索 → ヒアリング（「定義？計算式？使い方？」）
- **UI**: 選択肢ボタン表示、クリックで絞り込み

### 3. 小テスト解説
- **入力**: 「問題3が不正解だった」（+ 画像添付）
- **Agent動作**: 正答の根拠資料を検索 → 解説生成
- **UI**: 画像プレビュー + 解説テキスト + 根拠資料リンク

### 4. 明確な質問
- **入力**: 「決定係数の計算式は？」
- **Agent動作**: 直接回答 + 根拠提示
- **UI**: 回答テキスト + 根拠資料カード

---

## UI構成

### 1. 質問入力欄
- **必須要素**:
  - テキストエリア（Markdown対応、`TextField` from MUI）
  - 画像添付ボタン（小テスト画像、資料スクショ）
  - 送信ボタン（`Button` from MUI）
- **オプション**（Phase 3以降）:
  - 音声入力ボタン
  - 科目選択ドロップダウン（デフォルトは現在閲覧中の科目）

### 2. 推論状態表示
SSEイベントごとにUI更新：

| SSEイベント | UI表示 | コンポーネント |
|-----------|--------|------------|
| `thinking` | 「考えています...」 + ローディングアニメーション | `CircularProgress` (MUI) |
| `searching` | 「資料を探しています...」 + プログレスバー（`current_retry / max_retries`） | `LinearProgress` (MUI) |
| `evidence` | 根拠資料カード表示（下記） | `EvidenceCard` (shared/ui) |
| `answer` | 回答テキストを逐次表示（Markdownレンダリング） | `ReactMarkdown` |
| `done` | 入力欄を再度有効化 | - |
| `error` | エラーメッセージ + 再試行ボタン | `Alert` (MUI) + `Button` |

### 3. 根拠資料カード（Evidence Card）
`shared/ui/EvidenceCard` として実装。

**必須要素**:
- **資料名**: 例「統計学講義資料.pdf」
- **ページ番号**: 例「p.15」
- **セクション名**: 例「3.2 決定係数の定義」
- **クリッカブルリンク**: GCS署名付きURL + `#page=15`（新しいタブで開く）
- **選定理由**（`why_relevant`）: 例「計算式が記載されているため」（`Tooltip`で表示）

**デザイン**:
- MUI `Card`ベース
- ホバー時に影を強調（`elevation={3}` → `elevation={6}`）
- クリック時にブラウザの新しいタブでPDFを開く

**参照**: `../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`（情報階層: 根拠 → 要点 → 次の行動）

### 4. 質問履歴（スレッド形式）
- **表示内容**:
  - 過去の質問・回答を時系列で表示
  - 各エントリにタイムスタンプ、質問テキスト、回答サマリー
- **操作**:
  - クリックで過去の回答を再表示（キャッシュから取得、再リクエストなし）
  - 削除ボタン（ローカルのみ削除、サーバーには保持）

---

## 非機能要件

### アクセシビリティ
- **WCAG 2.1 AA準拠**
- スクリーンリーダー対応（`aria-label`, `aria-live`）
- キーボードナビゲーション（Tab順序の最適化）
- 詳細: `../../01_architecture/ACCESSIBILITY.md`

### レスポンシブ
- **デスクトップ**: 2カラムレイアウト（左: ファイルツリー、右: QAチャット）
- **モバイル**: 1カラム、ファイルツリーはドロワーに格納
- **Chrome拡張機能ポップアップ**: 最小幅320px対応

### パフォーマンス
- **SSEチャンク受信**: 60fps維持（React 18のConcurrent Rendering活用）
- **長文回答**: 仮想スクロール（`react-window`使用検討）
- **画像添付**: 最大5MB、圧縮処理（`browser-image-compression`）

---

## 実装例（疑似コード）

```typescript
// features/qa-chat/ui/QAChatPanel.tsx
'use client';

import { useState } from 'react';
import { Box, TextField, Button, CircularProgress } from '@mui/material';
import { EvidenceCard } from '@/shared/ui/EvidenceCard';
import { useQAStream } from '../model/useQAStream';

export function QAChatPanel({ subjectId }: { subjectId: string }) {
  const [question, setQuestion] = useState('');
  const { stream, send, isLoading } = useQAStream(subjectId);

  return (
    <Box>
      {/* 実装 */}
    </Box>
  );
}
```

---

## 参照
- FSD Slices定義: `../../01_architecture/SLICES_MAP.md`
- ブランドガイドライン: `../../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md`
- トーン&マナー: 落ち着いて、正確で、学習者に敬意のある表現
- エラーハンドリング: `../../03_integration/ERROR_HANDLING.md`
