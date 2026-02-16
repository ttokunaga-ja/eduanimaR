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
| `session` | ユーザーセッション | `useSession`, `SessionBadge` | Phase 2（SSO認証後） |

---

## Features（ユーザー価値のある機能）

| Slice | 責務 | 主要エクスポート | 備考 |
|-------|------|--------------|------|
| `qa-chat` | **汎用質問対応**（資料収集、質問明確化、小テスト解説すべて含む） | `QAChatPanel`, `useQAStream` | すべてのユースケースを単一UIで実現 |
| `auth-sso` | SSO認証（Google/Meta/Microsoft/LINE） | `SSOLoginButton` | Phase 2 |

**重要**: 
- 「資料検索機能」「質問精緻化機能」といった個別Featureは作らない。すべて`qa-chat`の責務。
- `qa-chat`は、SSEイベント（thinking/searching/evidence/answer）を受信してUI状態を更新する単一コンポーネント。

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
