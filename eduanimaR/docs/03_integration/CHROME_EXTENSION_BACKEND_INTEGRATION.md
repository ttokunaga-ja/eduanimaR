# Chrome Extension ↔ Backend Integration（Contract）

このドキュメントは、eduanima+R の **Chrome 拡張（Phase 2〜）** と **Professor（Go）** の統合を、セキュリティと運用を壊さない形で契約化します。

対象：
- Phase 2（資料の自動検知・自動保存）
- Phase 4（小テスト HTML 解析の送信）
- Phase 5（ページ文脈からの科目/トピック推定と自動検索）

SSOT（契約の正）：
- Professor OpenAPI：`../../eduanimaR_Professor/docs/openapi.yaml`

関連：
- SSE 契約：`SSE_STREAMING.md`
- エラーの標準：`ERROR_HANDLING.md` / `ERROR_CODES.md`
- Web の Auth/Session：`AUTH_SESSION.md`

---

## 結論（Must）

- 拡張は **Bearer JWT** で Professor API にアクセスする（OpenAPI `bearerAuth`）
- 拡張は Web の Cookie セッションに依存しない（環境差・ドメイン制約のため）
- Host permissions / 収集データは **最小化** する（プライバシー/セキュリティ）
- DOM監視・小テスト解析は、ユーザーが意図しない情報収集にならないよう、対象範囲と保存範囲を契約で固定する

---

## 1) 拡張の構成（推奨）

- **Popup**：軽量な状態確認（ログイン、選択中の科目、直近の保存）
- **Sidepanel**：Q&A（チャット）とストリーミング表示
- **Content Script**：LMS DOM の監視・資料検知・小テスト HTML 解析
- **Background（Service Worker）**：認証、API通信、キューイング、再送（content script を薄くする）

通信は原則として：
- content script → background（message）
- background → Professor（HTTPS）
- background → sidepanel/popup（message）

---

## 2) 認証（Must）

Professor API は `Authorization: Bearer <JWT>` を要求します（OpenAPI 参照）。

### 2.1 方針

- 拡張は短命の access token を使う
- refresh が必要なら「再ログイン導線」を用意する（無限失敗リトライは禁止）

### 2.2 トークン保管

- 優先：`chrome.storage.session`（ブラウザ終了で消える）
- 必要時のみ：`chrome.storage.local`（永続）

禁止：
- 長期トークンの平文永続化
- 権限の過剰要求（不要な `identity` / host permissions）

### 2.3 ログインUX（最小要件）

- 未認証時は Sidepanel/Popup からログイン導線を提示
- 認証失敗（401）はトークン破棄 → 再ログイン導線へ

---

## 3) Phase 2：自動保存（Auto-Ingestion）

### 3.1 資料検知（content script）

- LMS 上のリンク/埋め込み/ダウンロード導線を監視し、PDF/PPT等の候補を検知する
- 収集するメタデータは最小限：
  - `url`（取得元）
  - `filename`（推定可）
  - `subject_id`（ユーザー指定 or 推定）
  - `detected_at`

### 3.2 アップロード（Professor）

Professor 契約：
- `POST /v1/materials`（`multipart/form-data`）
- `subject_id` は必須
- `202 Accepted` を返し、解析はバックエンドで非同期に進む

拡張の実装方針：
- 検知 → background がダウンロード/アップロードを実行
- 失敗時はリトライ（上限回数 + バックオフ）

---

## 4) Phase 2〜：LMS 上フローティングボタン（その場で質問）

- content script がページ上にアクションを提供
- 質問実行は Sidepanel を開き、チャットUIへ誘導（誤操作・情報露出を抑える）

---

## 5) Q&A（SSEストリーミング）

Professor 契約：

1) セッション開始：`POST /v1/questions`
- リクエスト：`subject_id` と `question`
- レスポンス：`request_id` と `events_url`（`202`）

2) SSE購読：`GET /v1/questions/{request_id}/events`
- `text/event-stream`

拡張では `EventSource` のヘッダー制約があるため、原則：
- `fetch` + ReadableStream で SSE をパース（`SSE_STREAMING.md`）

---

## 6) Phase 4：小テスト HTML 解析（契約方針）

バックエンド API が未整備の場合でも、収集/送信の最小契約を先に固定する。

### 6.1 収集する内容（最小化）

- 問題文
- 選択肢
- ユーザーの選択
- 正誤
- 試行時刻

禁止：
- LMS の個人情報（氏名/学籍番号等）を収集する
- 画面全体のHTMLを無差別に送信する（必要部分のみ抽出）

---

## 7) Phase 5：コンテキスト自動認識（契約方針）

- ページ内容から subject/topic を推定し、バックエンド検索の物理フィルタ（`subject_id`）を自動適用する
- 推定は誤りうるため、UI で「現在の科目スコープ」を常に明示し、ユーザーが修正できる

---

## 8) エラー処理（Must）

- HTTP 失敗は `ERROR_HANDLING.md` / `ERROR_CODES.md` の分類に従う
- 拡張はネットワーク断が多い前提で、再送/再接続を実装する
- ただし無限再試行はしない（ユーザーへ明示し、操作で回復できる導線を用意）

---

## 9) パーミッション（最小化）

- Host permissions は LMS ドメインに限定（ワイルドカード乱用禁止）
- `storage` は必要最小限
- `scripting` 等の強い権限は理由がある場合のみ
