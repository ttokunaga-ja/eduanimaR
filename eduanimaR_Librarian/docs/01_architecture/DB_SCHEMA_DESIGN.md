# DB_SCHEMA_DESIGN（Librarian 観点の SSOT）

## 結論
`eduanima-librarian` は **DB を持たない**。したがって本サービス内での DB スキーマ設計は **行わない**。

## 目的
「DB を持たない」という境界を崩さないために、**データ所有権（Owning Data）と入出力の最小契約**を明文化する。

## データ所有権
- **Professor（Go）** が以下を所有する:
  - DB / 検索インデックス（全文/ベクトル）
  - 取り込み/バッチ処理/インデックス更新
  - ドキュメントID・ページ番号・チャンクID等の正規化
- **Librarian（Python）** が所有するのは、リクエスト内の LangGraph state のみ（メモリ内）。永続化しない。

## Librarian が扱う「参照（Evidence）」の最小形
Librarian の出力は「どの資料のどこを見るべきか」を示す参照の集合であり、DB行やスキーマではない。

推奨する最小契約は、LLMに安定IDを見せないために「二層」に分ける。

### A) LLM 可視の検索候補（Professor → Librarian）
- `temp_index`: 検索候補の一時参照（整数、リクエスト内で一意）
- `text`: テキスト断片（LLM が評価する対象）
- （任意）`location_hint`: 人間が読める位置情報（例: "p.12" / "Slide 12"）

### B) 不透明な対応表（Professor が保持）
`temp_index` を、Professor が管理する安定参照へ対応付ける（Librarian からは参照するだけで、LLM には渡さない）。

安定参照の例（Professor 内部）:
- `document_id`: Professor が管理する安定ID（文字列）
- `pages`: ページ番号（整数配列、1始まり）
- （任意）`chunk_id`: 断片ID
- （任意）`source_uri`: 原典のURI/パス
- （任意）`score`: 検索スコア（参照用途のみ。Librarian は閾値を固定しない）

### C) Librarian の選定結果（Librarian → Professor）
- `selected_evidence`: `temp_index` の集合（必要なら `why` を付与）

この形にすることで、Librarian の推論（LLM）には安定IDを露出させず、Professor が最終的に引用（安定参照）へ変換できる。

## 禁止事項（MUST）
- Librarian での DB 接続・スキーマ管理・マイグレーション
- Librarian での「インデックス正」を作る行為（検索結果は常に Professor の応答を正とする）

## 関連
- `01_architecture/MICROSERVICES_MAP.md`
- `01_architecture/SYNC_STRATEGY.md`
- `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
