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

## Phase 1実装スコープ（詳細）

Phase 1では、以下のコンポーネント/機能を**完全実装**します。

### 必須実装コンポーネント

#### 1. `features/qa-chat/ui/QAChatPanel.tsx`（Web版・拡張機能共通）

**責務**: 質問入力 + SSE受信 + UI状態管理

**依存**:
- `packages/shared-api`（Orval生成Hook）
- `packages/shared-ui/EvidenceCard`
- MUI v6コンポーネント（TextField, Button, CircularProgress 等）

**実装例**:
```typescript
// packages/features/qa-chat/ui/QAChatPanel.tsx
'use client'; // Web版（Next.js）ではClient Component

import { useState } from 'react';
import { Box, TextField, Button, CircularProgress, LinearProgress, Alert } from '@mui/material';
import { EvidenceCard } from '@packages/shared-ui/evidence-card';
import { useQAStream } from '../model/useQAStream';

interface QAChatPanelProps {
  subjectId: string;
}

export function QAChatPanel({ subjectId }: QAChatPanelProps) {
  const [question, setQuestion] = useState('');
  const { events, send, isLoading, error, retry } = useQAStream(subjectId);

  const handleSend = () => {
    if (question.trim()) {
      send(question);
      setQuestion('');
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      {/* SSEイベント表示 */}
      {events.map((event, idx) => {
        switch (event.type) {
          case 'thinking':
            return <CircularProgress key={idx} size={24} />;
          case 'searching':
            return (
              <Box key={idx}>
                <p>資料を探しています...</p>
                <LinearProgress variant="determinate" value={(event.data.current_retry / event.data.max_retries) * 100} />
              </Box>
            );
          case 'evidence':
            return (
              <EvidenceCard
                key={idx}
                materialName={event.data.material_name}
                pageNumber={event.data.page_number}
                sectionName={event.data.section_name}
                url={event.data.url}
                whyRelevant={event.data.why_relevant}
                snippet={event.data.snippet}
              />
            );
          case 'answer':
            return <div key={idx} dangerouslySetInnerHTML={{ __html: event.data.markdown }} />;
          case 'error':
            return (
              <Alert key={idx} severity="error">
                {event.data.message}
                <Button onClick={retry}>再試行</Button>
              </Alert>
            );
          default:
            return null;
        }
      })}

      {/* 質問入力欄 */}
      <TextField
        multiline
        rows={3}
        value={question}
        onChange={(e) => setQuestion(e.target.value)}
        placeholder="質問を入力してください"
        disabled={isLoading}
      />
      <Button onClick={handleSend} disabled={isLoading || !question.trim()}>
        送信
      </Button>
    </Box>
  );
}
```

#### 2. `packages/shared-ui/evidence-card/EvidenceCard.tsx`

**責務**: 根拠資料カードの表示（クリッカブル、ツールチップ付き）

**Props**:
```typescript
interface EvidenceCardProps {
  materialName: string;      // 資料名（例: 「統計学講義資料.pdf」）
  pageNumber: number;        // ページ番号（例: 15）
  sectionName?: string;      // セクション名（例: 「3.2 決定係数の定義」）
  url: string;               // GCS署名付きURL（例: "https://storage.googleapis.com/..."）
  whyRelevant: string;       // 選定理由（例: 「計算式が記載されているため」）
  snippet: string;           // Markdown抜粋（例: "> 決定係数は..."）
}
```

**実装例**:
```typescript
import { Card, CardContent, Typography, Tooltip, Box } from '@mui/material';
import ReactMarkdown from 'react-markdown';

export function EvidenceCard({
  materialName,
  pageNumber,
  sectionName,
  url,
  whyRelevant,
  snippet
}: EvidenceCardProps) {
  return (
    <Card
      elevation={3}
      sx={{ cursor: 'pointer', '&:hover': { elevation: 6 } }}
      onClick={() => window.open(`${url}#page=${pageNumber}`, '_blank')}
    >
      <CardContent>
        <Typography variant="h6">{materialName}</Typography>
        <Typography variant="body2" color="text.secondary">
          p.{pageNumber} {sectionName && `- ${sectionName}`}
        </Typography>
        <Tooltip title={whyRelevant}>
          <Box sx={{ mt: 1 }}>
            <ReactMarkdown>{snippet}</ReactMarkdown>
          </Box>
        </Tooltip>
      </CardContent>
    </Card>
  );
}
```

#### 3. `features/qa-chat/model/useQAStream.ts`

**責務**: SSEストリーミングの受信・パース・状態管理

**依存**: EventSource API（または`@microsoft/fetch-event-source`）

**実装例**:
```typescript
import { useState, useCallback } from 'react';

interface SSEEvent {
  type: 'thinking' | 'searching' | 'evidence' | 'answer' | 'done' | 'error';
  data: any;
}

export function useQAStream(subjectId: string) {
  const [events, setEvents] = useState<SSEEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const send = useCallback((question: string) => {
    setIsLoading(true);
    setEvents([]);
    setError(null);

    const eventSource = new EventSource(
      `/api/qa/ask?subject_id=${subjectId}&question=${encodeURIComponent(question)}`
    );

    eventSource.addEventListener('thinking', (e) => {
      setEvents((prev) => [...prev, { type: 'thinking', data: JSON.parse(e.data) }]);
    });

    eventSource.addEventListener('searching', (e) => {
      setEvents((prev) => [...prev, { type: 'searching', data: JSON.parse(e.data) }]);
    });

    eventSource.addEventListener('evidence', (e) => {
      setEvents((prev) => [...prev, { type: 'evidence', data: JSON.parse(e.data) }]);
    });

    eventSource.addEventListener('answer', (e) => {
      setEvents((prev) => [...prev, { type: 'answer', data: JSON.parse(e.data) }]);
    });

    eventSource.addEventListener('done', () => {
      setIsLoading(false);
      eventSource.close();
    });

    eventSource.addEventListener('error', (e) => {
      setError('エラーが発生しました');
      setIsLoading(false);
      eventSource.close();
    });
  }, [subjectId]);

  const retry = useCallback(() => {
    // 最後の質問を再送信（実装詳細は略）
  }, []);

  return { events, send, isLoading, error, retry };
}
```

### 実装除外（Phase 2以降）

Phase 1では以下を**実装しません**:

- ❌ **ファイルアップロードUI（Web版）**: Phase 1では拡張機能の自動アップロードのみ、Phase 2以降も実装しない
- ❌ **ユーザー登録UI**: Phase 2でSSO認証実装後に対応
- ❌ **SSO認証フロー**: Phase 1は`dev-user`固定

### Chrome拡張機能（Phase 1実装必須）

Phase 1では、以下の拡張機能を**完全実装**します:

1. ✅ **LMS資料の自動検知**（MutationObserver）
   - DOM変更監視、資料リンク検出
   - 検出した資料をローカルストレージに一時保存

2. ✅ **自動アップロード**（Plasmo Messaging + Professor API）
   - Content Scripts → Background/Service Worker → Professor API (`POST /v1/materials/upload`)
   - アップロード状態表示（成功/失敗/進行中）

3. ✅ **質問機能（QAチャット）**
   - Sidepanel/Popup内で`QAChatPanel`を表示
   - SSEイベントのリアルタイム表示

4. ✅ **ローカル動作検証**
   - `npm run build:extension` → Chromeに手動読み込み
   - ローカルProfessor API（`http://localhost:8080`）に接続

**参照**:
- [`../00_quickstart/QUICKSTART.md`](../00_quickstart/QUICKSTART.md)（Chrome拡張機能セクション）
- [`../03_integration/API_GEN.md`](../03_integration/API_GEN.md)（Chrome拡張機能からのAPI通信）

---

## 参照
- FSD Slices定義: `../../01_architecture/SLICES_MAP.md`
- ブランドガイドライン: `../../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md`
- トーン&マナー: 落ち着いて、正確で、学習者に敬意のある表現
- エラーハンドリング: `../../03_integration/ERROR_HANDLING.md`
