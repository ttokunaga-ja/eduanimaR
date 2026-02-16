# QA Page Requirements（Chrome拡張機能版）

Last-updated: 2026-02-16

このドキュメントは、Chrome拡張機能におけるQA機能（質問・回答UI）の要件を定義します。

**参照**:
- [`../../01_architecture/SLICES_MAP.md`](../../01_architecture/SLICES_MAP.md) - `qa-chat` Feature定義
- [`../../02_tech_stack/STACK.md`](../../02_tech_stack/STACK.md) - Chrome拡張機能の技術詳細

---

## 1. UI起動・終了パターン

### 起動方法

1. **FABメニューから起動**
   - Moodle画面右下のPENアイコン（FABメニュー）をクリック
   - メニュー内の「AI質問」アイテムをクリック
   - サイドパネルが画面右端からスライドイン表示（0.3秒アニメーション）

### 終了方法

1. **サイドパネル左端の「>」ボタンをクリック**（主要）
   - サイドパネルが画面右端へスライドアウト（0.3秒アニメーション）
   - 状態はsessionStorageに保存

2. **FABメニューから「AI質問」を再クリック**（トグル動作）
   - サイドパネルが閉じる
   - 状態はsessionStorageに保存

---

## 2. サイドパネル仕様

### レイアウト

- **配置**: 画面右端固定（position: fixed）
- **幅**: 400px（デフォルト）、将来的にリサイズ可能
- **高さ**: 画面全体（top: 0, bottom: 0）
- **z-index**: 999999（最前面）
- **背景色**: white
- **影**: box-shadow: -4px 0 16px rgba(0,0,0,0.1)

### 開閉アニメーション

- **プロパティ**: transform: translateX(100%) ↔ translateX(0)
- **所要時間**: 0.3秒
- **イージング**: ease

### 内部構造

```
┌─────────────────────────────────────┐
│ [>] 閉じるボタン                      │
├─────────────────────────────────────┤
│                                     │
│  QAChatPanel                        │
│  - 質問入力欄                        │
│  - SSE受信表示                       │
│  - エビデンスカード                  │
│                                     │
│                                     │
│                                     │
│                                     │
└─────────────────────────────────────┘
```

### Moodleレイアウトへの影響

**パネル開時**:
- `body { margin-right: 400px }` を設定
- メインコンテンツの幅が自動調整される
- スクロールバーは維持

**パネル閉時**:
- `body { margin-right: 0 }` に戻す
- メインコンテンツが元の幅に戻る

---

## 3. 状態永続化仕様

### sessionStorageでの保存項目

```typescript
interface PanelState {
  isOpen: boolean;               // パネル開閉状態
  width: number;                 // パネル幅（将来のリサイズ対応）
  scrollPosition: number;        // スクロール位置
  conversationHistory: Message[]; // 会話履歴
  lastUpdated: string;           // 最終更新日時（ISO 8601）
}
```

### 保存タイミング

- パネル開閉時
- 質問送信時
- SSE受信時（エビデンス・回答受信）
- スクロール時（デバウンス: 500ms）

### 復元タイミング

- Content Script再実行時（ページリロード）
- SPAナビゲーション時（`turbo:load`イベント）
- DOM再構築時（MutationObserver検知）

### 復元処理フロー

```typescript
window.addEventListener('load', () => {
  const state = restorePanelState();
  
  if (state?.isOpen) {
    // 1. パネルを開く
    toggleSidePanel();
    
    // 2. スクロール位置を復元
    setTimeout(() => {
      const chatContainer = document.querySelector('#chat-container');
      if (chatContainer) {
        chatContainer.scrollTop = state.scrollPosition;
      }
    }, 100);
    
    // 3. 会話履歴を復元
    if (state.conversationHistory.length > 0) {
      renderConversationHistory(state.conversationHistory);
    }
  }
});
```

---

## 4. ページ遷移対応

### 通常遷移（ページ全体リロード）

**動作**:
1. Content Script再実行
2. sessionStorageから状態復元
3. パネル再作成（状態維持）

### SPAナビゲーション（Turbo等）

**動作**:
1. `turbo:load`イベントを検知
2. パネルが既に存在するか確認
3. 存在しなければ再作成（状態維持）

**実装例**:
```typescript
document.addEventListener('turbo:load', () => {
  const state = restorePanelState();
  
  if (state?.isOpen) {
    // パネルが既に存在するか確認
    if (!document.getElementById('eduanima-sidepanel')) {
      createSidePanel();
    }
  }
});
```

### DOM再構築（Ajax等）

**動作**:
1. MutationObserverでFABメニューの削除・再挿入を監視
2. 「AI質問」アイテムを再挿入
3. サイドパネルは維持（状態変更なし）

**実装例**:
```typescript
const fabObserver = new MutationObserver(() => {
  const fabMenu = document.querySelector('.float-button-menu');
  
  if (fabMenu && !fabMenu.querySelector('.eduanima-qa-menu-item')) {
    // 「AI質問」アイテムを再挿入
    insertQAMenuItem(fabMenu);
  }
});

fabObserver.observe(document.body, {
  childList: true,
  subtree: true,
});
```

---

## 5. QAChatPanel コンポーネント仕様

### 責務

- ユーザー入力の受付
- SSEイベント（thinking/searching/evidence/answer）の受信・表示
- エビデンスカードの表示（クリッカブルURL、why_relevant、snippets）
- 会話履歴の管理（sessionStorage永続化）

### 表示状態

| 状態 | 表示内容 |
|------|---------|
| **アイドル** | 質問入力欄のみ表示 |
| **thinking** | プログレス表示「AI Agentが検索方針を決定しています」 |
| **searching** | プログレスバー表示「資料を検索中... (2/5回目)」 |
| **evidence** | エビデンスカード表示（複数件） |
| **answer** | 回答テキストをリアルタイム表示 |
| **done** | 完了（再質問可能） |
| **error** | エラートースト表示 + 再試行ボタン |

### エビデンスカードの必須要素

- **クリッカブルpath/url**: GCS署名付きURLで原典にアクセス可能
- **ページ番号（page）**: 該当箇所のページ番号（例：「p.3」）
- **why_relevant**: なぜこの箇所が選ばれたかの説明文
- **snippets**: 資料からの抜粋（Markdown形式）
- **heading**: 該当セクションの見出し

---

## 6. 非機能要件

### パフォーマンス

- サイドパネル表示: 0.3秒以内
- SSE受信遅延: 100ms以内
- スクロール復元: 100ms以内

### アクセシビリティ

- キーボード操作対応（Esc キーでパネル閉じる）
- スクリーンリーダー対応（ARIA属性）

### ブラウザ対応

- Chrome 最新版（Manifest V3）
- Chromium系ブラウザ（Edge、Brave等）

---

## 7. Phase 1スコープ

### ✅ 実装すべきこと

1. FABメニュー統合（「AI質問」アイテム追加）
2. サイドパネル表示（幅400px、固定）
3. 状態永続化（sessionStorage）
4. ページ遷移対応（通常遷移・SPAナビゲーション）
5. QAChatPanel コンポーネント（SSE受信・エビデンス表示）

### ❌ 実装しないこと

1. パネルのリサイズ機能（Phase 2以降）
2. フォールバック（独立ボタン等）
3. Popup/新規タブ方式
4. 会話履歴の永続化（localStorage、Phase 2以降）

---

## 8. 参照

- FSDレイヤー定義: [`../../01_architecture/FSD_LAYERS.md`](../../01_architecture/FSD_LAYERS.md)
- `qa-chat` Feature定義: [`../../01_architecture/SLICES_MAP.md`](../../01_architecture/SLICES_MAP.md)
- Chrome拡張機能の技術詳細: [`../../02_tech_stack/STACK.md`](../../02_tech_stack/STACK.md)
- Professor OpenAPI契約: [`../../../eduanimaR_Professor/docs/openapi.yaml`](../../../eduanimaR_Professor/docs/openapi.yaml)
