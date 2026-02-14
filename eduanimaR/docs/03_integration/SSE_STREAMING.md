# SSE Streaming（Frontend Contract）

このドキュメントは、eduanima+R における **Server-Sent Events（SSE）** のフロントエンド実装・運用の契約を定義します。

SSOT（契約の正）：
- Professor OpenAPI（SSE含む）：`../../eduanimaR_Professor/docs/openapi.yaml`

関連：
- エラーの標準：`ERROR_HANDLING.md`
- エラーコード：`ERROR_CODES.md`
- 通信境界（SSE必須）：`../../eduanimaR_Professor/docs/03_integration/INTER_SERVICE_COMM.md`

---

## 結論（Must）

- Q&A のユーザー体験は **SSE でストリーミング** する（無反応時間を作らない）
- SSE イベントは **契約** として扱い、イベント名/ペイロード形は安定させる
- クライアントは **再接続** を前提に実装する（ネットワーク断・タブ復帰等）
- Unknown event は落ちずに無視できる（前方互換）

---

## 1) エンドポイント（契約）

Professor（Go）外向き API：

1. セッション開始
- `POST /v1/questions`
- `202 Accepted`
- レスポンス：`request_id` と `events_url`

2. SSE 購読
- `GET /v1/questions/{request_id}/events`
- `200 OK`
- `Content-Type: text/event-stream`

---

## 2) SSE フレーム形式

Professor OpenAPI 例：

```text
event: progress
data: {"type":"progress","stage":"searching","message":"Searching..."}

event: answer
data: {"type":"answer","delta_markdown":"..."}
```

### 2.1 取り扱いルール（Must）

- `data:` は **1行 JSON** として扱う（`JSON.parse` 可能であること）
- `event:` が無い場合でも `message` として処理できるようにする（実装は堅牢に）
- `answer` は **差分（delta）** を前提に UI を更新する（全文置換前提にしない）

---

## 3) イベント種別（推奨）

OpenAPI では `progress` / `answer` が例示されています。
フロントでは、最低限以下の「分類」を想定して実装します（Unknown は無視）。

### 3.1 `progress`

用途：進捗メッセージ（例：検索中、要約中、整形中）

推奨ペイロード：
```json
{
  "type": "progress",
  "stage": "searching",
  "message": "Searching..."
}
```

### 3.2 `answer`

用途：回答本文のストリーミング

推奨ペイロード：
```json
{
  "type": "answer",
  "delta_markdown": "..."
}
```

備考：Source（根拠リンク）を「いつ」送るかは実装次第ですが、フロントは以下の両方を許容します。
- 末尾で「まとめて」届く
- 途中で段階的に届く

---

## 4) 再接続（Must）

SSE はネットワーク断・スリープ復帰で切断されます。クライアントは以下を満たすこと。

- UI 状態（入力、既出の delta）は保持する
- 切断時は「再接続中」を表示する（黙って止めない）
- 再接続はバックオフ（指数 + ジッタ）

注意：Professor OpenAPI では `Last-Event-ID` 等の厳密な再開契約は定義されていません。
そのため現時点のフロント方針は以下です。

- 再接続は **同じ request_id の events** を購読し直す
- サーバーが重複イベントを返す可能性を想定し、フロント側で idempotent に扱う

---

## 5) エラー/終了（Must）

### 5.1 HTTP エラー

`/events` が `401/403/404` 等を返す場合は、`ERROR_HANDLING.md` / `ERROR_CODES.md` の分類に従って UI を出し分けます。

最低限：
- `401`：再認証導線
- `403`：権限なし表示
- `404`：request_id が無い/期限切れ → 再送（新規質問）導線

### 5.2 ストリームの自然終了

ストリームが閉じたら、以下を行う：
- UI を「完了」に遷移
- 追加送信（追質問）が可能なら入力欄を有効化

---

## 6) 実装ガイド（推奨）

### 6.1 Web（Next.js）

- SSE はブラウザ `EventSource` が基本。ただし `Authorization: Bearer ...` を付けたい場合は、
  - **同一オリジン + Cookie セッション** に寄せる（Web）
  - もしくは `fetch` + ReadableStream で `text/event-stream` を自前パースする

どちらを採用するかは、認証方式（Cookie/JWT）とデプロイ構成（BFF有無）で確定する。

### 6.2 Chrome 拡張

- 拡張は Bearer JWT を前提に `fetch` ストリームパース方式を推奨（EventSource はヘッダー制約がある）
- content script ではなく、必要に応じて service worker（background）で通信して UI に転送する

---

## 禁止（AI/人間共通）

- イベント名/JSON形をフロント都合で変更してバックエンドに押し付ける（契約破壊）
- ストリーミング中に例外を握りつぶして UI を止める（再接続/エラー表示へ）
- Unknown event でクラッシュする（前方互換を壊す）
