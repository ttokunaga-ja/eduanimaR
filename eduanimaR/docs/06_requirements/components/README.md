# Component Requirements（Index）

このフォルダは、再利用するコンポーネントの要件を管理します。

- テンプレ：`COMPONENT_REQUIREMENTS_TEMPLATE.md`
- 新規作成時はテンプレを複製して埋めます

推奨命名：`C_<id>_<ComponentName>_REQUIREMENTS.md`

---

## コンポーネント要件テンプレート（情報階層に基づく）

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

---

## エビデンス表示コンポーネント要件

### Professor OpenAPI契約の必須要素

エビデンスカードコンポーネント（`entities/evidence/ui/EvidenceCard`）は、以下の要素を含む必要があります：

**必須表示要素**:
- **クリッカブルpath/url**: GCS署名付きURLで原典にアクセス可能
- **ページ番号（page）**: 該当箇所のページ番号（例：「p.3」）
- **why_relevant**: なぜこの箇所が選ばれたかの説明文
- **snippets**: 資料からの抜粋（Markdown形式）
- **heading**: 該当セクションの見出し

**実装要件**:
- エビデンスカードは「主役」として画面上部に配置（情報階層に基づく）
- クリック時に原典（PDF/GCSリンク）へ遷移
- why_relevantを明示し、学習者が「なぜ」を理解できるようにする

**データ構造例**:
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

## チャットUIコンポーネント要件

### トーン&マナー（Handbookより）

チャットUIコンポーネント（`features/qa-chat/ui/ChatMessage`）は、以下のトーン&マナーに基づきます：

**Voice（不変の声）**:
- 落ち着いて（Calm）、正確で（Accurate）、学習者に敬意がある（Respectful）
- 結論より根拠を示す（Show rationale over conclusions）
- 次の一歩を短く、複雑さを避ける（Keep next step short, avoid complexity）

**UI Copy Rules（UI文言のルール）**:
- 結論を先に、その後に根拠
- 専門用語を最小化
- 失敗時は次のステップを表示
- 共有・削除の影響を明示

**プログレスフィードバック**:
- 技術用語をユーザーに見せない（「Librarian推論実行中」→「AI Agentが資料を検索中です」）
- ライトユーザー向けの簡潔な表現を使用
- プログレスバーまたはスピナーで視覚的にフィードバック

**参照**: [`../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md`](../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md)
