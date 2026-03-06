# MUI_PIGMENT

## 目的
MUI v6 + Pigment CSS の採用方針と制約を定義し、フロントエンド（Next.js）とバックエンド（Go）の連携において UI パフォーマンスを最大化する。

## Pigment CSS とは
- **ゼロランタイム (Zero-runtime) CSS**: ビルド時にスタイルを生成し、ブラウザで実行時に JS が動いてスタイルを計算することを避ける
- **利点**: Core Web Vitals（LCP/FID/CLS）の改善、初回描画速度の向上
- **注意**: 仕様/実装が更新中のため、アップグレード時の破壊的変更に注意

## DO（推奨）
- `shared/ui/` 配下に MUI のラッパーコンポーネントを作成し、Pigment CSS のスタイル設定を集約する
- 静的なスタイル（色・サイズ・余白）はビルド時に確定させる
- テーマ設定（`theme.ts`）で色・フォント・ブレークポイントを定義し、全体で統一する

## DON'T（禁止）
- 上位層（features/widgets/pages）で直接 `sx` prop を多用しない（動的スタイルはランタイムコストが高い）
- MUI の API を直接 import せず、必ず `shared/ui/` 経由で使う
- Emotion（MUI v5以前）の `styled` API を混在させない

## 制約（バックエンドとの関係）
- バックエンド（Go）が返すデータ構造は、フロントエンドの表示ロジックに影響する
- 例: 回答に添付する Source の種類（GCS URL / LMS URL / ファイルパス）に応じて、フロントエンドは表示（クリック可能、ページ番号等）を事前に定義しておく必要がある
- この対応関係は `ERROR_CODES.md` と同様に、バックエンド・フロントエンド間で同期させる

## アップグレード時の確認観点
- Pigment CSS の Breaking Changes を確認（MUI の公式リリースノート）
- `sx` prop の動作が変わっていないか
- Next.js との統合（`next.config.js` の設定）に変更がないか

## バックエンド側の技術標準（対比）
- `GO_1_25_GUIDE.md`: Go の標準ライブラリ優先（`log/slog`, `errors` 等）
- `SQLC_QUERY_RULES.md`: ORM 禁止、sqlc による型生成
- `ECHO_HANDLERS.md`: Echo v5 のハンドラー実装ルール
- `ELASTICSEARCH_OPS.md`: Post-MVP（MVPでは使用しない）

