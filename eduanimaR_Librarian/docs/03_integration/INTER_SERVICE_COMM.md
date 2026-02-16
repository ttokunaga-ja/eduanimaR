# INTER_SERVICE_COMM（Librarian SSOT）

## 目的
Professor（Go）↔ Librarian（Python）の通信規約を定義し、責務境界（DB-less/推論特化）を壊さずに統合できるようにする。

## 通信構成（確定）
```
eduanima-professor (Go)  ↔  eduanima-librarian (Python)
         gRPC/Proto (双方向ストリーミング)
         契約SSOT: eduanimaR_Professor/proto/librarian/v1/librarian.proto

eduanima-librarian (Python)  →  Gemini API
               HTTPS
```

## Professor ↔ Librarian
### プロトコル
- **gRPC/Proto（双方向ストリーミング）** を正とする。
- 契約の SSOT は Professor 側の `proto/librarian/v1/librarian.proto`。
- Phase 3 検索ループにおいて、Professor と Librarian 間で複数ターン双方向通信を実現するために gRPC を採用。

### 認証/認可
- 原則「内部サービス間通信」。認証方式（mTLS/Workload Identity 等）は運用側 SSOT に従う。
- Librarian は最終回答文を生成しないため、認可の主戦場は Professor 側（利用者/セッション文脈の管理）。

### タイムアウト/リトライ
- timeout は必須。gRPC の deadline/cancellation を適切に扱う。横断ルールは `01_architecture/RESILIENCY.md` を正とする。
- retry はネットワーク起因の一時失敗に限定し、推論ループ（MaxRetry）とは分離する。

### 検索ツール呼び出し（責務境界）
- Librarian は **検索クエリ**を生成し、Professor の検索ツールへ渡す。
- **取得件数（k）/除外（既出断片）/候補集合の規模（N）/試行回数**などの「物理検索の状態」は Professor が所有する。
- Professor は必要に応じて「広く取得 → 多様性/重複の抑制 → Librarian に渡す候補を絞る」を行う（Librarian は方針を固定しない）。

### メタデータの扱い（LLM 非露出の原則）
- Librarian の推論（LLM）に **安定ID（document_id 等）を見せない**。
- Professor → Librarian の検索候補は、LLM 可視の `temp_index` + テキスト断片を基本とし、安定参照への対応表は Professor が保持する。

### エラー
- エラー形式/コードは `ERROR_HANDLING.md`, `ERROR_CODES.md` に従う。

## 契約（SSOT）
- gRPC/Proto 運用: `API_CONTRACT_WORKFLOW.md`
- 契約定義: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
- 破壊的変更の扱い: `API_VERSIONING_DEPRECATION.md`

## 明確に「やらない」こと
- Librarian が DB/検索基盤へ直接アクセスすること
- Librarian がイベント同期（CDC/Outbox 等）を操作すること
