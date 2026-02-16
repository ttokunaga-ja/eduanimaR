# MUI v6 + Pigment CSS Style Guide

我々はランタイムパフォーマンスを最大化するため、Pigment CSS を使用します。

注意：Pigment CSS は開発が活発で、破壊的変更や制約の更新が起き得ます。導入/アップグレード時は MUI 公式の Pigment CSS ガイドも必ず確認してください。

## 前提（確定スタック）
本テンプレートの確定版技術スタックは [STACK.md](./STACK.md) を参照。

---

## デザイン原則（Handbookより）

eduanimaRのデザインは、以下の4原則に基づきます：

1. **Calm & Academic（落ち着いた学術的雰囲気）**
   - 過度なアニメーションを避ける
   - 学習に集中できる落ち着いた配色
   - 装飾より可読性を優先
2. **Clarity First（明瞭性優先）**
   - 情報の階層を明確にする
   - タイポグラフィの一貫性を保つ
   - 専門用語を最小化
3. **Trust by Design（信頼できる設計）**
   - データの共有範囲を明示
   - 権限が曖昧にならない
   - 誤って他者のデータが見えることがない
4. **Evidence-forward（エビデンスを主役に）**
   - 根拠となる資料を常に明示
   - ソース（資料名・ページ番号）を先頭に表示
   - クリッカブルなリンクで原典にアクセス可能

**参照**: [`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)、[`../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md`](../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md)

## 色の使用方針（Handbookより）

eduanimaRの色使用は、以下の原則に基づきます：

- **色は意味に紐づける**: 装飾目的での色の増殖を避ける
  - Neutral（グレースケール）: デフォルト
  - Accent（ブランドカラー）: アクションボタン
  - Success/Warning/Error: 状態フィードバック
- **重要な警告は色だけに依存しない**: アイコンやテキストを併用
- **アクセシビリティ**: WCAG AA準拠のコントラスト比を確保

**参照**: [`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)

## タイポグラフィ原則（Handbookより）

eduanimaRのタイポグラフィは、以下の原則に基づきます：

- **可読性優先**: 行間・文字間を適切に設定
- **階層の明確化**: h1-h6、body、captionで情報階層を表現
- **一貫性**: フォントサイズ・ウェイトを統一
- **和文対応**: 日本語フォントの可読性を考慮

**参照**: [`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)

---

## セットアップ方針（Next.js）

Pigment CSS はビルド時にスタイルを抽出します。Next.js ではプラグイン設定と stylesheet の import が必要です。

- `next.config.*`：`withPigment` を適用
- `src/app/layout.tsx`：`@pigment-css/react/styles.css` を import

（具体例はプロジェクトの実装に合わせて記述・更新する）

## ✅ 推奨パターン (DO)

### 0. 原則：静的に書く（ゼロランタイム志向）
- `styled` / `css` は **静的な object** で書く
- 条件分岐が必要なら **variants（列挙）** か **CSS変数** で表現する

### 1. `styled` の使用
```tsx
import { styled } from '@mui/material-pigment-css';

const CustomButton = styled('button')(({ theme }) => ({
  backgroundColor: theme.palette.primary.main,
  padding: theme.spacing(2),
}));
```

補足：MUIコンポーネント以外の汎用スタイル（`css`/`globalCss`/`keyframes`）が必要な場合は `@pigment-css/react` の API を使う。

### 2. 静的な `sx` プロパティ
```tsx
<Box sx={{ display: 'flex', flexDirection: 'column' }}>...</Box>
```

`sx` はビルド時に `className` / `style` へ変換されます。乱用すると「その場しのぎのデザイン」と「差分追跡困難」を招くため、原則は `shared/ui` のラッパー側に閉じ込めます。

### 3. css変数の活用 (動的値が必要な場合)
```tsx
const DynamicBox = styled('div')({
  color: 'var(--box-color)',
});

// Component内
<DynamicBox style={{ '--box-color': props.color } as React.CSSProperties} />
```

### 4. variants（列挙値による条件付きスタイル）
実行時に自由入力の値でスタイルを変えるのではなく、`size`/`variant` のような列挙に寄せます。

```tsx
import { styled } from '@mui/material-pigment-css';

export const Badge = styled('span')({
  padding: '2px 8px',
  borderRadius: 999,
  variants: [
    { props: { tone: 'success' }, style: { backgroundColor: 'var(--success-bg)' } },
    { props: { tone: 'danger' }, style: { backgroundColor: 'var(--danger-bg)' } },
  ],
});
```

## 🚫 禁止パターン (DON'T)

1. **Emotionの使用**: `import styled from '@emotion/styled'` は禁止。
2. **無制限なランタイム動的スタイル**: スタイル定義内で任意の値（例：ユーザー入力）を直接受け取って色/サイズ等を変える。まず variants か CSS変数で解決する。
3. **`makeStyles` / `withStyles`**: これらは廃止されました。

## Runtime theme（注意）

`useTheme` 等で参照できる runtime theme は、シリアライズ可能な値のみで構成され、モード（ライト/ダーク）で値が変化しない場合があります。

- 原則：見た目を theme 値で切り替える場合は `theme.vars.*`（CSS変数）を参照する
- runtime theme は本当に必要な場合だけ使う

## Agent への最重要指示
- Pigment CSS を使うファイルは原則 `*.tsx`
- 「見た目だけ直す」でも、禁則（Emotion, makeStyles, 無制限なランタイム動的スタイル）に触れないこと
