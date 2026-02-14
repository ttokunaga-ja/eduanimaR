# SKILL: Tailwind CSS（Scoped Styles / Shadow DOM）

対象：Web版とChrome拡張でTailwind CSSを安全に使用し、LMSサイトへの影響を防ぐ。

変化に敏感な領域：
- Chrome拡張の Content Script でのスタイル隔離
- Shadow DOM の制約
- Tailwind の設定（プリフライト無効化等）

関連：
- `../03_integration/CHROME_EXTENSION_BACKEND_INTEGRATION.md`
- `../02_tech_stack/MUI_PIGMENT.md`
- `../05_operations/SUPPLY_CHAIN_SECURITY.md`

---

## Versions（2026-02-15）

- `tailwindcss`: `^3.4.0`
- `postcss`: `^8.4.0`
- `autoprefixer`: `^10.4.0`

---

## Must

- Web版: 通常の Tailwind CSS 使用（スコープ不要）
- Chrome拡張: Shadow DOM で隔離し、LMSサイトへの影響を防ぐ
- Tailwind の Preflight（リセットCSS）を Content Script では無効化
- カスタムプレフィックス（例: `tw-`）を使用してクラス名衝突を防ぐ（拡張のみ）

---

## Web版の設定（通常）

```javascript
// tailwind.config.js
export default {
  content: [
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      // カスタムテーマ
    },
  },
  plugins: [],
};
```

```css
/* globals.css */
@tailwind base;
@tailwind components;
@tailwind utilities;
```

---

## Chrome拡張の設定（Shadow DOM + Preflight無効化）

### 1) Preflight 無効化

```javascript
// extension/tailwind.config.js
export default {
  content: [
    './extension/src/**/*.{js,ts,jsx,tsx}',
  ],
  corePlugins: {
    preflight: false, // Preflight（リセットCSS）を無効化
  },
  prefix: 'tw-', // すべてのクラスに `tw-` プレフィックス
  theme: {
    extend: {},
  },
};
```

### 2) Shadow DOM で隔離

```typescript
// extension/src/content/inject.tsx
import { createRoot } from 'react-dom/client';
import styles from './styles.css?inline'; // CSSを文字列として読み込み

// Shadow DOM を作成
const shadowHost = document.createElement('div');
shadowHost.id = 'eduanimar-extension-root';
document.body.appendChild(shadowHost);

const shadowRoot = shadowHost.attachShadow({ mode: 'open' });

// Shadow DOM 内にスタイルを注入
const styleElement = document.createElement('style');
styleElement.textContent = styles;
shadowRoot.appendChild(styleElement);

// React ルートを Shadow DOM 内にマウント
const appContainer = document.createElement('div');
shadowRoot.appendChild(appContainer);

const root = createRoot(appContainer);
root.render(<App />);
```

### 3) Vite 設定（インラインCSS）

```typescript
// extension/vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        // CSS を JS にインライン化
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'content.css';
          }
          return 'assets/[name]-[hash][extname]';
        },
      },
    },
  },
});
```

---

## LMS サイトへの影響を防ぐ

### 1) Shadow DOM の利点

- LMS サイトのスタイルと完全に隔離される
- Tailwind の Utility クラスが LMS に影響しない
- LMS のグローバルスタイルが拡張に影響しない

### 2) プレフィックスの追加（二重対策）

Shadow DOM だけでは不十分な場合、プレフィックスを追加：

```javascript
// tailwind.config.js (拡張用)
export default {
  prefix: 'tw-', // `tw-flex`, `tw-p-4` 等
  // ...
};
```

React コンポーネント：

```tsx
// ❌ Before
<div className="flex p-4 bg-white">

// ✅ After
<div className="tw-flex tw-p-4 tw-bg-white">
```

---

## テーマの共通化（Web / 拡張）

Web版と拡張でテーマを共通化する場合：

```javascript
// shared/tailwind-theme.js
export const theme = {
  colors: {
    primary: '#3B82F6',
    secondary: '#10B981',
    // ...
  },
  spacing: {
    // ...
  },
};
```

```javascript
// web/tailwind.config.js
import { theme } from '../shared/tailwind-theme';

export default {
  theme: {
    extend: theme,
  },
};
```

```javascript
// extension/tailwind.config.js
import { theme } from '../shared/tailwind-theme';

export default {
  corePlugins: {
    preflight: false,
  },
  prefix: 'tw-',
  theme: {
    extend: theme,
  },
};
```

---

## 禁止

- Chrome拡張で Preflight（リセットCSS）を有効にする（LMSスタイルを破壊）
- Shadow DOM を使わずに Content Script でスタイルを注入する
- Web版と拡張で異なるテーマを使う（一貫性が失われる）
- グローバルセレクタ（`body`, `html`）を使う（LMSに影響）

---

## チェックリスト

- [ ] Chrome拡張で Preflight が無効化されているか？
- [ ] Shadow DOM でスタイルが隔離されているか？
- [ ] プレフィックス（`tw-`）がすべてのクラスに付いているか？（拡張のみ）
- [ ] CSS がインライン化されているか？（拡張のみ）
- [ ] Web版と拡張でテーマが共通化されているか？
- [ ] グローバルセレクタを使用していないか？

---

## トラブルシューティング

### 拡張のスタイルが適用されない

1. **Shadow DOM 内にスタイルが注入されているか確認**
   - DevTools で Shadow Root を開き、`<style>` タグがあるか確認

2. **CSS がインライン化されているか確認**
   - Vite の `?inline` クエリが正しく設定されているか

3. **Preflight が無効化されているか確認**
   - `tailwind.config.js` で `preflight: false` が設定されているか

### LMS サイトのスタイルが壊れる

1. **Shadow DOM が使われているか確認**
   - Shadow DOM なしでスタイルを注入していないか

2. **Preflight が有効になっていないか確認**
   - Preflight はグローバルリセットCSSなので、無効化が必須

---

## 参考

- Tailwind CSS: https://tailwindcss.com/
- Shadow DOM: https://developer.mozilla.org/en-US/docs/Web/API/Web_components/Using_shadow_DOM
- Chrome Extensions: https://developer.chrome.com/docs/extensions/
