# Resiliency（Frontend / BFF）

このドキュメントは、フロントエンド（Next.js BFF + Browser）におけるレジリエンスを
“実装の自由” にせず、運用可能な契約として固定します。

対象：
- Browser（Client Components）
- Next.js（RSC / Route Handler / Server Action）
- upstream（Go Gateway / Microservices）

関連：
- エラー分類：`../03_integration/ERROR_HANDLING.md`
- エラーコード：`../03_integration/ERROR_CODES.md`
- 観測性：`../05_operations/OBSERVABILITY.md`

---

## 結論（Must）

- タイムアウト・再試行・キャンセルを“方針化”して乱立させない
- リトライは **無条件にしない**（副作用のある操作は idempotency を前提に）
- 二重送信（double submit）を防ぐ（UI/サーバ双方）
- ユーザーが復帰できるUX（再試行/再読み込み/状態保持）を用意する

---

## 1) タイムアウト

- upstream 呼び出しは必ず上限時間を持つ
- タイムアウト時は `UPSTREAM_TIMEOUT` 相当へ正規化し、UIで再試行可能にする

推奨：
- GET 系：短め（体感に直結）
- POST/PUT 系：少し長め（ただし無限に待たない）

---

## 2) リトライ（Retry）

### リトライしてよい
- GET など安全な読み取り
- 429/503/504 など一時障害

### リトライに条件が必要
- mutation（POST/PUT/DELETE）
  - 原則：idempotency-key がある場合のみ自動リトライ可
  - 無い場合：ユーザー操作での明示再試行（ボタン）に寄せる

### eduanimaR固有：Librarian推論失敗時の扱い

- Librarian が停止条件を満たせず終了した場合（`event: error`、`reason: insufficient_evidence`）
  - **自動リトライしない**（コスト・時間考慮）
  - UIに「情報が不足しています。質問を具体化するか、資料を追加してください」と表示
  - 「再試行」ボタンを提供（ユーザー判断）

- Professor/Librarian のタイムアウト
  - SSE接続: 60秒（推論ループ最大時間）
  - タイムアウト時は接続を切断し、「処理に時間がかかっています。もう一度お試しください」と表示

---

## 3) キャンセル（Abort）

- 画面遷移・入力変更で不要になったリクエストはキャンセルできる設計にする
- 連打/多重実行を抑止する（in-flight の状態管理）

---

## 4) 二重送信防止（Idempotency / UI）

- UI：送信中はボタン無効化・スピナー等で状態を明示
- Server（BFF）：可能なら idempotency-key を付与・透過する
- upstream が冪等でない操作は、安易に自動再試行しない

---

## 5) フォールバック（Fallback）

- 部分的に壊れても全体を落とさない（widgets 単位のフォールバック等）
- ただし “重要情報の欠落” を成功扱いしない（観測に載せる）

---

## 6) 観測（運用の前提）

- timeout / retry / cancel の発生回数・比率を観測できる状態にする
- requestId/traceId を運べるなら運ぶ（Next→Go の遅延分解が可能になる）

---

## 7) 禁止（AI/人間共通）

- 無限リトライ
- mutation の自動リトライ（冪等性なし）
- catchして握りつぶし（運用で検知できない）
