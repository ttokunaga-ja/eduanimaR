# PROJECT_DECISIONS（eduanimaR_Professor固有の決定事項）

Owner: @ttokunaga-ja  
Status: Published  
Last-updated: 2026-02-17  
Tags: professor, backend, decisions

---

## 1. プロジェクトの性質
eduanimaR_Professor（Go バックエンド）は、**学習効果検証のための研究プロジェクト**として位置づける。

- 収益化は研究完了後の検討事項（Phase 1〜4では非営利）
- Phase 1の主目的は「技術的実現可能性の検証」と「学習効果の測定」

---

## 2. Phase 1のスコープ固定（SSOT）

### 提供する機能
- 資料アップロード（PDF/PowerPoint → GCS保存）
- OCR + 構造化（Gemini 2.0 Flash使用）
- pgvector埋め込み生成・保存
- Q&A API（単一科目内検索 + 根拠提示）

### スコープ外
- SSO認証（dev-user固定）
- 複数ユーザー対応
- Kafka非同期処理（Phase 1は同期処理のみ）
- Elasticsearch（Phase 1はpgvectorのみ）

---

## 3. 技術的決定事項

### データベース
- PostgreSQL 18.1 + pgvector 0.8.1
- マイグレーション管理: Atlas v1.0.0
- クエリ生成: sqlc 1.30.0
- ドライバ: pgx v5.8.0

### 外部API
- OCR/構造化: Gemini 2.0 Flash
- 埋め込み生成: Gemini Embedding（768次元）

### デプロイ
- Phase 1: ローカル実行のみ（Docker Compose）
- Phase 2以降: Google Cloud Run

---

## 4. OpenAPI契約（Phase 1版）

### 必須エンドポイント
1. `POST /subjects/{subjectId}/materials` - 資料アップロード
2. `POST /qa` - 質問応答
3. `GET /materials/{materialId}/status` - 処理状態確認

### 契約の配置
- SSOT: `eduanimaR_Professor/docs/openapi.yaml`
- 生成先（Frontend）: `eduanimaR/src/shared/api/` （Orval自動生成）

---

## 5. 研究データ収集方針

### 取得するデータ
- OCR精度（文字認識率、処理時間）
- 検索応答時間（p50/p95/p99）
- ユーザーフィードバック（根拠の有用性5段階評価）

### 倫理的配慮
- 個人を特定可能なデータは取得しない
- 学習行動データは匿名化して研究利用
- 被験者への事前説明と書面同意を必須化

---

## 6. Phase 1の完了条件

1. 検索成功率70%以上（10件の検証質問で7件成功）
2. 検索応答時間p95で5秒以内
3. ハルシネーション率20%以下
4. 5名以上の被験者から肯定的評価

上記を達成した場合のみ、Phase 2（SSO認証+複数ユーザー）へ移行する。
