# Slices Map（FSDスライス定義）

Last-updated: 2026-02-16

このドキュメントは、eduanimaRフロントエンドの各FSDレイヤーにおけるスライス（Slice）定義のSSOTです。

## 目的
- スライスの追加/変更時に必ず参照・更新する
- AI/人間が同じ判断基準でスライス配置を決定できるようにする

---

## Entities（ビジネス実体）

| Slice | 責務 | 主要エクスポート | 備考 |
|-------|------|--------------|------|
| `subject` | 科目（Professor の subject_id に対応） | `SubjectCard`, `useSubject` | 科目一覧表示、選択UI |
| `file` | 資料ファイル（GCS URL / metadata） | `FileCard`, `useFile` | 資料メタデータ表示 |
| `evidence` | エビデンス（資料の根拠箇所） | `EvidenceCard`, `useEvidence` | SSE `evidence` イベントを受信・表示。資料名・ページ・抜粋・why_relevantを含む |
| `session` | ユーザーセッション | `useSession`, `SessionBadge` | Phase 2（SSO認証後） |

---

## Features（ユーザー価値のある機能）

| Slice | 責務 | 主要エクスポート | 備考 |
|-------|------|--------------|------|
| `qa-chat` | **汎用質問対応**（すべてのユースケースを単一UIで実現） | `QAChatPanel`, `useQAStream` | **SSEイベント処理**:<br>- `thinking`: 「検索戦略を立案中...」プログレスバー<br>- `searching`: 「資料を検索中...（試行 X/5）」プログレスバー<br>- `evidence`: 資料カード表示（資料名・ページ・抜粋・why_relevant）<br>- `answer`: 回答ストリーミング表示（Markdown形式）<br>- `complete`: Good/Badフィードバックボタン表示<br>- `error`: エラーコード別UI表示（`ERROR_CODES.md`参照）<br><br>**フィードバック機能**（Phase 1）:<br>- 回答完了後、Good/Badボタンを表示<br>- フィードバック送信後、お問い合わせフォーム（Googleフォーム）へのリンクを表示<br><br>**対応ユースケース**:<br>- 資料収集依頼<br>- 質問内容の明確化<br>- 小テスト解説<br>- 明確な質問への直接回答<br><br>**提供チャネル**:<br>- **Chrome拡張**: MoodleのFABメニューから起動 → サイドパネル表示（画面右端、幅400px、リサイズ可能、状態永続化）<br>- **Webアプリ**: ページ内に常時表示（Phase 2以降）<br><br>**重要**: 個別のFeature（`search-materials`、`clarify-question`など）は作らない。すべて`qa-chat`の責務。 |
| `auth-sso` | SSO認証（Google/Meta/Microsoft/LINE） | `SSOLoginButton` | Phase 2 |

---

### `qa-chat` のユースケース例（すべて単一UIで実現）

以下のユースケースは、すべて `features/qa-chat` の単一UIで実現されます：

| ユースケース | Professor Phase 2の判断 | フロントエンドのUI動作 |
|------------|----------------------|-------------------|
| **資料収集依頼**<br>「統計学の資料を集めて」 | 収集戦略決定 | SSEで `searching` イベント受信 → 資料一覧をエビデンスとして表示 |
| **曖昧な質問**<br>「決定係数って何？」 | ヒアリング優先 | SSEで `thinking` イベント受信 → Phase 4-Aで意図候補3つ表示 → ユーザー選択 |
| **小テスト解説**<br>「問題3の答えが間違ってた」 | 解答根拠検索 | SSEで `searching` → `evidence` イベント受信 → 正答の根拠資料を表示 |
| **明確な質問**<br>「決定係数の計算式は？」 | 検索実行 | SSEで `searching` → `evidence` → `answer` イベント受信 → 直接回答 + エビデンス表示 |

**重要な設計原則**:
- ❌ **禁止**: 「資料検索機能」「質問精緻化機能」といった個別Featureを作る
- ✅ **正解**: すべて`qa-chat`の単一UIで実現し、SSEイベントに応じて表示を切り替える
- **理由**: バックエンド（Professor Phase 2）が戦略を判断するため、フロントエンドは「質問を投げてSSEで受け取る」だけでよい

**参照**: 
- [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md)（Phase 2の検索 vs ヒアリング判断）
- [`../README.md`](../README.md)（AI Agent質問システムの柔軟性）

---

## Widgets（複数Feature/Entityの合成）

| Slice | 責務 | 使用するSlice | 備考 |
|-------|------|-------------|------|
| `file-tree` | 科目別ファイルツリー表示 | `entities/subject`, `entities/file` | サイドバーに配置 |
| `qa-layout` | QA画面全体のレイアウト | `features/qa-chat`, `widgets/file-tree` | Phase 1 |

---

## Pages（画面の組み立て）

| Slice | 責務 | ルート | 備考 |
|-------|------|-------|------|
| `home` | ホーム画面（科目一覧） | `/` | Phase 1 |
| `qa` | QA画面（汎用質問対応） | `/qa` | Phase 1（メイン画面） |
| `auth` | 認証画面（SSO選択） | `/auth/login` | Phase 2 |

---

## Shared（共通部品）

### `shared/api`
- Professor OpenAPI 生成クライアント（Orval）
- エラーハンドリング共通処理

### `shared/ui`
- MUI Pigment ベースの共通コンポーネント（Button, TextField, Card等）
- Evidence Card（根拠資料表示カード）

### `shared/lib`
- 日時フォーマット、文字列処理等のユーティリティ
- i18n設定（Phase 2以降）

---

## スライス追加のガイドライン

### 追加前の確認
1. **既存スライスに含められないか？**（特に`features/qa-chat`）
2. **上位レイヤーでの合成で済まないか？**（Widgetsで合成）
3. **Sharedに落とせる汎用処理ではないか？**

### 追加時の手順
1. このファイル（SLICES_MAP.md）を更新
2. `eslint-plugin-boundaries`の設定を更新（`.eslintrc.js`）
3. Public API（`index.ts`）を作成
4. PR説明に「スライス追加の理由」を明記

---

## 参照
- FSDレイヤー定義: [FSD_LAYERS.md](./FSD_LAYERS.md)
- FSD概要: [FSD_OVERVIEW.md](./FSD_OVERVIEW.md)
- バックエンド境界: [DATA_ACCESS_LAYER.md](./DATA_ACCESS_LAYER.md)
