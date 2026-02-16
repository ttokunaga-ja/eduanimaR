# Component Architecture（UI設計の責務境界）

Last-updated: 2026-02-16

このドキュメントは、FSD（Feature-Sliced Design）を前提に、UIコンポーネントの責務と配置を固定します。

目的：
- UI部品が pages に閉じ込められる/肥大化するのを防ぐ
- 状態・副作用が UI に漏れて「どこで何が起きるか」を曖昧にしない
- 生成API / DAL / キャッシュ戦略と矛盾しない設計にする

関連：
- レイヤー：`FSD_LAYERS.md`
- DAL：`DATA_ACCESS_LAYER.md`
- SSR/Hydration：`../02_tech_stack/SSR_HYDRATION.md`

---

## コンポーネント設計原則（情報階層とデザイン原則）

### 情報階層（Handbookより）

eduanimaRのUI設計は、以下の情報階層に基づきます：

1. **主役：根拠（Evidence）**
   - 資料名、ページ番号、セクション、抜粋
   - クリッカブルなpath/url
   - why_relevant（なぜこの箇所が選ばれたか）
2. **次点：要点（Key Points）**
   - 箇条書き形式
   - 学習者が理解すべきポイント
3. **行動：次の一歩（Next Action）**
   - 復習すべき箇所
   - 次に学ぶべき関連トピック
   - 関連資料の探索

**参照**: [`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)

### UI/UX要件（デザイン4原則）

eduanimaRのデザインは、以下の4原則に基づきます：

1. **Calm & Academic**: 落ち着いた学術的な雰囲気
   - 過度なアニメーションを避ける
   - 学習に集中できる落ち着いた配色
2. **Clarity First**: 可読性を装飾より優先
   - 情報の階層を明確にする
   - タイポグラフィの一貫性を保つ
3. **Trust by Design**: 権限が曖昧にならない設計
   - データの共有範囲を明示
   - 誤って他者のデータが見えることがない
4. **Evidence-forward**: ソースが主役
   - 根拠となる資料を常に明示
   - クリッカブルなリンクで原典にアクセス可能

**参照**: [`../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md`](../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md)、[`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)

### エビデンス表示コンポーネント要件

Professor OpenAPI契約に基づく必須要素：

- **クリッカブルpath/url**: GCS署名付きURLまたは資料へのリンク
- **ページ番号**: 該当箇所のページ番号（例：「p.3」）
- **why_relevant**: なぜこの箇所が選ばれたかの説明文
- **snippets**: 資料からの抜粋（Markdown形式）
- **heading**: 該当セクションの見出し

**実装例（参考）**:
```typescript
interface EvidenceCardProps {
  documentId: string;
  path: string; // クリッカブルURL
  page: number;
  heading: string;
  snippets: string[];
  whyRelevant: string;
}
```

**参照**: [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md)、[`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md)

---

## 結論（Must）

- “画面合成” は `pages` / `widgets` が責務（features/entities を寄せ集める）
- “ユーザー価値の操作” は `features` が責務（フォーム、mutation、分岐）
- “ドメイン実体の表示” は `entities` が責務（表示と最小ロジック）
- “再利用UIプリミティブ” は `shared/ui`（ビジネスルール禁止）
- API 由来の生データを UI に丸渡ししない（DTO最小化はDAL/adapterで）

---

## プログレスフィードバックのパターン

### ❌ 禁止: 技術用語をユーザーに見せる
以下のような技術用語は UI に表示しない：
- 「検索クエリ生成中」
- 「Librarian推論実行中」
- 「ベクトル検索実行中」
- 「RRFスコア計算中」

### ✅ 推奨: ライトユーザー向けの簡潔な表現
以下のような分かりやすい表現を使用：
- 「AI Agentが資料を検索中です」
- 「AI Agentが検索方針を決定しています」
- 「AI Agentが回答を生成しています」

### 段階的なフィードバック例

```typescript
// プログレスフィードバックの段階例
const progressStages = [
  'AI Agentが検索方針を決定しています',
  'AI Agentが資料を検索中です',
  'AI Agentが回答を生成しています',
];
```

### ビジュアルフィードバック
- プログレスバーまたはスピナーで視覚的にフィードバック
- 進捗状況を数値で表示（例: 「3/5」）
- 完了予測時間は表示しない（不正確になりやすい）

**参照元SSOT**:
- `../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md` (Voice & Tone)
- `../../eduanimaRHandbook/04_product/ROADMAP.md` (UI/UXコンセプト)

---

## 1) コンポーネント種別と置き場所

### shared/ui（UI primitives / wrappers）
- 例：Button、Input、Dialog、Table、Layout primitives
- 特徴：
  - 見た目/アクセシビリティ/テーマに責務を限定
  - ビジネスルール・API呼び出しは禁止

### entities/*/ui（Entity presentation）
- 例：`UserCard`、`ProductPrice`、`OrderStatusBadge`
- 特徴：
  - entity の表示に必要な最小ロジック
  - 他entityやfeatureの合成は禁止

### features/*/ui（Use case UI）
- 例：`LoginForm`、`AddToCartButton`、`UpdateEmailForm`
- 特徴：
  - 入力 → 検証 → mutation → 成功/失敗 の分岐
  - API hooks の利用はここ（もしくは features/model）

### widgets/*/ui（Composite blocks）
- 例：`AppHeader`、`SearchResultsPanel`
- 特徴：
  - 複数features/entitiesの合成
  - ページ固有の事情を持ちにくい“塊”

### pages/*/ui（Page composition）
- 例：`UserProfilePage`
- 特徴：
  - ルーティングに対応する画面（URL）
  - 画面専用の薄い整形（formatting）まで

---

## 2) Props（DTO）設計の原則

- Client Component に渡す props は **最小**（表示に必要なフィールドのみ）
- バックエンドレスポンス型（生成型）を、そのまま UI props にしない
- 画面都合の整形は `pages/widgets` の adapter か `shared/api/dal` で行う

関連：`DATA_ACCESS_LAYER.md`

---

## 3) State の置き場所（目安）

- Server state：TanStack Query（`features` / `entities` の model で）
- UI local state：コンポーネント内（ただし再利用するなら model へ）
- Cross-feature の “なんでもストア”：原則禁止（依存地獄になりやすい）

---

## 4) Server/Client 境界（RSC 時代）

- 可能な限り RSC で表示を完結させ、不要な client 化を避ける
- client 化する理由（操作がある、ブラウザAPIが必要等）を明文化する

---

## 5) レビュー観点（チェックリスト）

- UIが適切な層に置かれている（pages肥大化していない）
- props が過剰でない（DTO最小化できている）
- shared/ui にビジネスルールが入っていない
- features が entities/widgets/pages を直接抱え込んでいない
