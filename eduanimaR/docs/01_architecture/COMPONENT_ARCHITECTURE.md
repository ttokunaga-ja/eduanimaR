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

## eduanimaR固有のUI設計原則（Must）

### 情報階層の厳守（Evidence-forward）

すべてのQA回答UIは、以下の順序で情報を配置する:

#### 1. 根拠（Evidence）- 主役

**必須要素**:
- 資料名、ページ番号、セクション名
- クリッカブルなGCS署名付きURL
- `why_relevant`（なぜこの箇所が選ばれたか）
- 抜粋は引用として視覚的に区別（`blockquote`など）

**視覚的優先度**: 最も目立つ位置・スタイルで配置

#### 2. 要点（Key Points）- 次点

**必須要素**:
- 箇条書き形式で学習者が理解すべきポイントを提示
- 断定より根拠・前提を示す表現

**視覚的優先度**: 根拠の次に配置、読みやすいタイポグラフィ

#### 3. 次の一歩（Next Action）- 行動

**必須要素**:
- 復習すべき箇所、関連トピック、関連資料の探索
- 複雑さを増やさず短く提示

**視覚的優先度**: 最後に配置、アクションを促す控えめなスタイル

---

### UI実装の悪い例・良い例

#### ❌ 悪い例（情報階層が逆転）

```tsx
<Card>
  <Typography variant="h6">決定係数の説明</Typography>
  <Typography>決定係数は回帰分析の説明力を示す指標です。</Typography>
  <Typography variant="caption">参考: 統計学テキスト p.45</Typography>
</Card>
```

**問題点**:
- 根拠（資料名・ページ・抜粋）が主役になっていない
- `why_relevant` が欠落
- クリッカブルなリンクがない

#### ✅ 良い例（情報階層を厳守）

```tsx
<Card>
  <Box sx={{ mb: 2 }}>
    <Typography variant="subtitle2" fontWeight="bold">
      📚 根拠資料
    </Typography>
    <Link href={evidenceUrl} target="_blank" sx={{ fontSize: '1.1rem' }}>
      統計学テキスト（p.45）
    </Link>
    <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
      なぜこの資料か: 決定係数の定義と計算式が明示されています
    </Typography>
    <blockquote style={{ borderLeft: '3px solid #ccc', paddingLeft: '1rem' }}>
      決定係数（R²）は、回帰モデルの説明力を0〜1の範囲で示す指標です。
    </blockquote>
  </Box>
  
  <Box sx={{ mb: 2 }}>
    <Typography variant="subtitle2" fontWeight="bold">
      📝 要点
    </Typography>
    <ul>
      <li>決定係数は回帰分析の説明力を示す</li>
      <li>0〜1の範囲で、1に近いほど説明力が高い</li>
    </ul>
  </Box>
  
  <Box>
    <Typography variant="subtitle2" fontWeight="bold">
      🔍 次の一歩
    </Typography>
    <Typography variant="body2">
      関連トピック: 相関係数、回帰分析の前提
    </Typography>
  </Box>
</Card>
```

**正しい点**:
- 根拠が最も目立つ位置・スタイルで配置
- クリッカブルなリンク（GCS署名付きURL）
- `why_relevant` の明示
- 抜粋が引用として視覚的に区別されている

---

### トーン&マナー（UI表現の原則）

すべてのUI文言は、以下の原則に基づき設計する:

1. **落ち着いて正確**: パニックを煽らない、事実ベースで伝える
2. **敬意のある表現**: 学習者に対して丁寧で前向きな言葉遣い
3. **次の行動を示す**: 「エラーです」で終わらず、解決策を提示

**参照**: 
- [`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)（情報階層）
- [`../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md`](../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md)（トーン&マナー）

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
- 「gRPC通信中」
- 「Phase 2実行中」

### ✅ 推奨: ライトユーザー向けの簡潔な表現

Phase別の推奨表示文言:

```typescript
// Professor SSE progressイベントのstageに応じた表示
const progressMessages = {
  planning: 'AI Agentが質問を理解しています',      // Phase 2
  searching: 'AI Agentが資料を検索中です',         // Phase 3
  finalizing: 'AI Agentが回答を生成しています',    // Phase 4-B
};

// SSEイベントハンドリング例
eventSource.addEventListener('progress', (event) => {
  const { stage } = JSON.parse(event.data);
  setProgressMessage(progressMessages[stage] || 'AI Agentが処理中です');
});
```

### Phase別の処理内容と表示文言

| Phase | Professor責務 | Librarian責務 | Frontend表示 | SSOT参照 |
|-------|--------------|--------------|-------------|----------|
| **Phase 2** | 検索 vs ヒアリング判断<br>検索戦略決定 | - | 「AI Agentが質問を理解しています」 | `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md` |
| **Phase 4-A** | 意図推測モード<br>候補3つ生成 | - | 意図選択UI表示<br>（Phase 3移行せず） | `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md` |
| **Phase 2再実行** | 選択意図をコンテキストに<br>検索戦略再決定 | - | 「AI Agentが質問を理解しています」 | `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md` |
| **Phase 3** | Librarian gRPC通信<br>権限強制<br>ハイブリッド検索（RRF統合） | 戦略に基づくクエリ生成<br>推論ループ（最大5回） | 「AI Agentが資料を検索中です」 | `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`<br>`../../eduanimaR_Librarian/docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md` |
| **Phase 4-B** | 最終回答モード<br>回答生成<br>SSE配信 | - | 「AI Agentが回答を生成しています」 | `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md` |

### 意図選択UI（Phase 4-A）

**Phase 2でヒアリング判断された場合のフロー**:

```typescript
// SSE clarificationイベント受信
eventSource.addEventListener('clarification', (event) => {
  const { question, intents } = JSON.parse(event.data);
  // intents: 厳密に3要素の配列
  // [
  //   { id: "intent-1", summary: "〜の資料を探したい" },
  //   { id: "intent-2", summary: "〜の概念を理解したい" },
  //   { id: "intent-3", summary: "〜の問題の解き方を知りたい" }
  // ]
  
  showIntentSelectionUI(question, intents);
});
```

**UI表示例**:
```
┌─────────────────────────────────────────┐
│ AI Agent                                │
├─────────────────────────────────────────┤
│                                         │
│ どの内容について知りたいですか？          │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ 線形代数の固有値の資料を探したい        │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ 固有値の概念を理解したい              │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ 固有値の計算問題の解き方を知りたい      │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ ─────────────────────────────────────  │
│ 上記にない場合は再度入力してください       │
│ [                              ] [送信] │
└─────────────────────────────────────────┘
```

**ユーザーアクション後のフロー**:
- **選択肢クリック**: `POST /v1/question/refine` → Phase 2再実行 → Phase 3 → Phase 4-B
- **テキスト再入力**: `POST /v1/question` → Phase 2から再実行

**重要な設計原則**:
- **候補数**: 3つ固定（Chrome拡張の表示範囲制約）
- **confidence表示**: なし（学習者信頼維持、Handbookブランドガイドライン準拠）
- **会話履歴**: previousRequestID で紐付け保持（追跡可能性、North Star Metric計測）

### ビジュアルフィードバック
- プログレスバーまたはスピナーで視覚的にフィードバック
- Phase 3の繰り返し（最大5回試行）は「検索中」のまま（進捗数値表示なし）
- 完了予測時間は表示しない（不正確になりやすい）

**参照元SSOT**:
- `../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md` (Voice & Tone)
- `../../eduanimaRHandbook/04_product/ROADMAP.md` (UI/UXコンセプト)
- `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md` (Phase責務詳細)

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
