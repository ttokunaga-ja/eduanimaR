# DB_SCHEMA_DISCUSSION

## 目的
[DB_SCHEMA_TABLES.md](./DB_SCHEMA_TABLES.md) で提案したスキーマ設計に対する議論ポイント・オープンクエスチョン・実装前の意思決定事項を整理する。

> 本ファイルは設計レビューでの議論を促進し、意思決定を文書化するための作業ドキュメント。

## 前提
- PostgreSQL 18.1 + pgvector 0.8.1 + Atlas + sqlc
- MVP（Minimum Viable Product）を前提とした設計
- 将来拡張を考慮しつつ、過剰設計を避ける

---

## 🔴 Critical: 実装前に必ず決める事項

### 1. ID戦略の実装詳細

**問題**:
- UUIDv7の生成方法（PostgreSQL 18.1の `uuidv7()` 関数 vs Go側生成）
- NanoIDの生成方法（Go側生成 + DB挿入 vs PostgreSQL関数）

**提案**:
```sql
-- UUIDv7: PostgreSQL 18.1標準関数を使用（推奨）
id UUID PRIMARY KEY DEFAULT uuidv7()

-- NanoID: Go側で生成してINSERT時に渡す（推奨）
-- 理由: PostgreSQL拡張に依存せず、テスト・モックが容易
```

**決定事項** (要記入):
- [ ] UUIDv7生成: `PostgreSQL uuidv7()` / `Go github.com/gofrs/uuid` / その他
- [ ] NanoID生成: `Go側` / `PostgreSQL拡張` / その他
- [ ] NanoIDの長さ: デフォルト21文字 / カスタム長（例: 12文字）

---

### 2. ベクトル次元数の確定

**問題**:
- Embeddingモデル（例: Vertex AI Embeddings, OpenAIなど）の選定と次元数
- 提案では `vector(768)` としているが、実際のモデルに合わせる必要がある

**候補**:
- Vertex AI `textembedding-gecko@003`: 768次元
- OpenAI `text-embedding-3-small`: 1536次元
- OpenAI `text-embedding-3-large`: 3072次元

**決定事項** (要記入):
- [ ] Embeddingモデル: ________________
- [ ] ベクトル次元数: ________________
- [ ] HNSWパラメータの初期値: `m=____`, `ef_construction=____`

---

### 3. マルチテナント・アクセス制御の詳細

**問題**:
- MVP時点での共有範囲（個人のみ vs 科目単位で共有）
- 将来的な共有機能の実装方針

**シナリオ**:
1. **個人利用のみ（MVP推奨）**: 
   - `materials.user_id` = アップロードユーザー固定
   - 科目内でも資料は共有しない
   
2. **科目単位で共有（将来拡張）**:
   - `subject_users` テーブル追加
   - ロール管理（owner/instructor/student）
   - 資料の閲覧権限管理

**決定事項** (要記入):
- [ ] MVP範囲: `個人利用のみ` / `科目内共有あり`
- [ ] 将来的な共有機能の要否: `要` / `不要` / `保留`
- [ ] `subject_users` テーブルの追加タイミング: `初期` / `Phase 2` / `不要`

---

### 4. 全文検索の言語設定

**問題**:
- PostgreSQL全文検索の辞書設定（`to_tsvector('english', ...)` vs その他）
- 日本語/多言語対応の要否

**候補**:
- `english`: 英語のステミング（推奨: MVP）
- `simple`: 言語非依存（日本語も検索可能だが精度は低い）
- pg_bigm / pg_trgm: N-gram検索（日本語対応）
- 外部検索エンジン（Elasticsearch等、将来拡張）

**決定事項** (要記入):
- [ ] MVP言語設定: `english` / `simple` / `pg_bigm` / その他
- [ ] 日本語対応の優先度: `高` / `中` / `低` / `不要`
- [ ] 多言語対応の方針: ________________

---

### 5. Chunk分割戦略

**問題**:
- チャンクサイズ（文字数/トークン数）の基準
- チャンク分割のロジック（Gemini 3 Flashの指示）

**考慮事項**:
- LLMのコンテキストウィンドウ（Gemini 3 Pro: 2M tokens）
- 検索精度（小さすぎると文脈不足、大きすぎると関連性低下）
- Embedding API の制限（例: 最大2048トークン/リクエスト）

**提案**:
- **チャンクサイズ**: 300〜1000文字（約100〜300トークン）
- **分割単位**: 意味段落（Gemini 3 Flashで自動判定）
- **オーバーラップ**: 前後50文字（文脈保持）

**決定事項** (要記入):
- [ ] 目標チャンクサイズ: ______〜______ 文字
- [ ] 分割単位: `意味段落` / `固定文字数` / `ページ単位` / その他
- [ ] オーバーラップ: `有り (____文字)` / `無し`

---

### 6. ジョブ管理とKafka連携

**問題**:
- `ingest_jobs` テーブルとKafkaトピックの関係
- 冪等性キーの生成方法（UUID vs その他）

**提案**:
```go
// 冪等性キー生成（例）
idempotencyKey := fmt.Sprintf("ingest-%s-%d", materialID, currentVersion)
```

**決定事項** (要記入):
- [ ] 冪等性キー形式: ________________
- [ ] Kafkaトピック名: ________________
- [ ] DLQ（Dead Letter Queue）の要否: `要` / `不要`
- [ ] リトライ戦略: `指数バックオフ` / `固定間隔` / その他

---

## 🟡 Important: 早めに決めたい事項

### 7. ページ情報の管理粒度

**問題**:
- `material_pages` テーブルの必要性
- ページレベルのOCR結果を保存するか、チャンクに統合するか

**選択肢**:

**A) ページテーブルあり（提案）**:
```
材料 → ページ → チャンク
```
- 利点: ページ単位の再処理が容易、OCR結果の保持
- 欠点: テーブルが増える、JOIN回数増加

**B) ページテーブルなし**:
```
材料 → チャンク（ページ範囲を保持）
```
- 利点: シンプル、JOINが減る
- 欠点: ページ単位の操作が煩雑

**決定事項** (要記入):
- [ ] 選択: `A (ページテーブルあり)` / `B (ページテーブルなし)`
- [ ] 理由: ________________

---

### 8. 世代管理（Generation/Version）の戦略

**問題**:
- LLMモデル更新時の再生成方針
- 旧世代データの保持期間

**シナリオ**:
- Gemini 4がリリースされたら、全資料を再処理する？
- 旧世代のチャンク/Embeddingは削除 vs 保持？

**提案**:
```sql
-- 最新世代のみをアクティブ化
UPDATE chunks SET is_active = FALSE WHERE material_id = :id AND generation < :new_gen;
INSERT INTO chunks (..., generation = :new_gen) VALUES (...);
```

**決定事項** (要記入):
- [ ] 再生成トリガー: `手動` / `自動（モデル更新検知）` / `バッチ処理`
- [ ] 旧世代データ保持: `削除` / `アーカイブ (is_active=false)` / `別テーブル移行`
- [ ] 世代管理の粒度: `資料単位` / `チャンク単位` / その他

---

### 9. セッション履歴の保存期間

**問題**:
- `reasoning_sessions` の保存期間（無制限 vs 定期削除）
- プライバシー・ストレージコストの考慮

**提案**:
- MVP: 無期限保存（ソフトデリートのみ）
- 将来: 90日後にアーカイブ or 削除（GDPR等考慮）

**決定事項** (要記入):
- [ ] MVP保存期間: `無期限` / `90日` / その他
- [ ] アーカイブ戦略: `ソフトデリート` / `別DB移行` / `完全削除`

---

### 10. インデックス戦略の詳細

**問題**:
- 複合インデックスの順序（例: `(subject_id, user_id)` vs `(user_id, subject_id)`）
- WHERE句でのフィルタ条件との整合性

**分析**:
```sql
-- クエリパターン1: 科目内の全ユーザー資料（管理者向け）
SELECT * FROM materials WHERE subject_id = :subject_id;

-- クエリパターン2: ユーザーの特定科目資料（学生向け、MVP主経路）
SELECT * FROM materials WHERE subject_id = :subject_id AND user_id = :user_id;
```

**推奨**:
- MVP: `(subject_id, user_id)` 優先（学生の個人利用想定）
- 将来: 管理者向けクエリが増えたら `(subject_id)` 単独インデックス追加

**決定事項** (要記入):
- [ ] 複合インデックス順序: `(subject_id, user_id)` / `(user_id, subject_id)` / 両方
- [ ] パーシャルインデックス: `WHERE is_active` を全インデックスに付与？

---

## 🟢 Nice-to-have: 議論すると良い事項

### 11. JSONBの活用範囲

**問題**:
- `material_pages.metadata` のスキーマ（自由形式 vs 型定義）
- `reasoning_sessions.plan_json` の検証

**提案**:
```json
// material_pages.metadata の例
{
  "chapter_title": "第3章 データベース設計",
  "figures": [{"id": "Fig3.1", "caption": "ER図"}],
  "tables": [{"id": "Table3.1", "caption": "比較表"}]
}
```

**決定事項** (要記入):
- [ ] JSONBスキーマの型定義: `アプリ側バリデーション` / `PostgreSQL CHECK制約` / `不要`
- [ ] インデックス: `GIN (全体)` / `特定パス (jsonb_path_ops)` / `不要`

---

### 12. 監査ログの粒度

**問題**:
- 現在の設計では `created_at` / `updated_at` のみ
- より詳細な監査（誰が・いつ・何を変更したか）の要否

**選択肢**:
1. **最小限（提案）**: `created_at`, `updated_at` のみ
2. **中程度**: `created_by`, `updated_by` カラム追加
3. **完全**: 専用 `audit_logs` テーブルで全変更を記録

**決定事項** (要記入):
- [ ] MVP監査レベル: `1 (最小限)` / `2 (中程度)` / `3 (完全)`
- [ ] 将来拡張の優先度: `高` / `中` / `低`

---

### 13. ソフトデリートの運用

**問題**:
- `is_active` + `deleted_at` の運用方針
- 物理削除（GDPR等での完全削除要求）への対応

**提案**:
```sql
-- ソフトデリート
UPDATE materials SET is_active = FALSE, deleted_at = NOW() WHERE id = :id;

-- 物理削除（管理者のみ、30日後など）
DELETE FROM materials WHERE is_active = FALSE AND deleted_at < NOW() - INTERVAL '30 days';
```

**決定事項** (要記入):
- [ ] ソフトデリート保持期間: `無期限` / `30日` / その他
- [ ] 物理削除の自動化: `要（バッチ）` / `手動のみ` / `不要`

---

### 14. パフォーマンステスト計画

**問題**:
- ベンチマークの目標値（検索レイテンシ、スループット）
- 負荷テストのシナリオ

**提案指標**:
- **検索レイテンシ**: 
  - 全文検索 < 50ms (p95)
  - ベクトル検索 < 100ms (p95)
- **スループット**: 100 req/sec（単一科目）
- **データ規模**: 科目あたり1000ファイル、10万チャンク

**決定事項** (要記入):
- [ ] 目標レイテンシ（全文検索）: ______ ms (p95)
- [ ] 目標レイテンシ（ベクトル検索）: ______ ms (p95)
- [ ] 負荷テストツール: `k6` / `JMeter` / `Gatling` / その他
- [ ] テスト実施タイミング: `MVP前` / `MVP後` / `Phase 2`

---

### 15. スキーマ進化（Schema Evolution）戦略

**問題**:
- 破壊的変更（カラム削除、ENUM値削除）への対応
- Expand/Contract パターンの適用方針

**提案**:
```
# Expand/Contract パターン
1. Expand: 新カラム追加（NULL許容 or デフォルト値）
2. Migrate: 旧→新へデータ移行（バックグラウンド）
3. Contract: 旧カラム削除（全移行完了後）
```

**決定事項** (要記入):
- [ ] Expand/Contract適用: `全変更` / `破壊的変更のみ` / `不要`
- [ ] ダウンタイム許容: `ゼロダウンタイム必須` / `短時間メンテ可` / `未定`

---

## 次のステップ

### 実装開始前のチェックリスト
- [ ] Critical事項（1〜6）の全決定完了
- [ ] Important事項（7〜10）の方針確定
- [ ] Nice-to-have事項（11〜15）の優先度確認
- [ ] Atlas `schema.hcl` のドラフト作成
- [ ] sqlc設定ファイル準備
- [ ] Testcontainersテスト環境構築

### レビュー・承認
- [ ] DB設計レビュー実施（チーム/ステークホルダー）
- [ ] セキュリティレビュー（物理制約の妥当性）
- [ ] パフォーマンス要件の合意
- [ ] 本ドキュメント (DB_SCHEMA_DISCUSSION.md) への決定事項記入

---

## 議論履歴

### 議論ログ（日付・参加者・決定事項を記録）

#### 2026-02-15: 初回設計案提示
- **参加者**: [記入]
- **議題**: DB_SCHEMA_TABLES.md の初回レビュー
- **決定事項**:
  - [記入]
- **次回アクション**:
  - [記入]

---

## 関連ドキュメント
- [DB_SCHEMA_TABLES.md](./DB_SCHEMA_TABLES.md) - テーブル定義（SSOT）
- [DB_SCHEMA_DESIGN.md](./DB_SCHEMA_DESIGN.md) - 設計原則
- [MICROSERVICES_MAP.md](./MICROSERVICES_MAP.md) - サービス境界
- [STACK.md](../02_tech_stack/STACK.md) - 技術スタック
- [SKILL_DB_ATLAS_SQLC_PGX.md](../skills/SKILL_DB_ATLAS_SQLC_PGX.md) - 実装ガイド
