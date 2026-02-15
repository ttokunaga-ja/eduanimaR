# Error Codes（Frontend Mapping）

このドキュメントは、バックエンドが返す失敗を **フロントの挙動へ一意にマッピング**するための契約です。

目的：
- 画面ごとの例外処理の乱立を防ぐ
- エラーを分類し、UXと運用（観測）を一致させる

関連：
- エラー処理：`ERROR_HANDLING.md`
- 観測性：`../05_operations/OBSERVABILITY.md`

---

## 結論（Must）

- エラーは **code** を中心に扱う（message 文字列で分岐しない）
- 未知の code は **Unknown** として安全側に倒す（error boundary / 汎用メッセージ）
- UI挙動はこの表をSSOTとし、実装の分岐は一箇所に寄せる（例：`shared/api/errors.ts`）

---

## 推奨コード体系（テンプレ）

プロジェクトのバックエンド体系に合わせて調整してください。
最低限、以下のカテゴリが区別できること。

| Category | code（例） | HTTP（目安） | UI挙動 | 再試行 | 運用通知 |
| --- | --- | --- | --- | --- | --- |
| AuthN | `UNAUTHORIZED` | 401 | ログイン導線/再認証 | × | △（急増で） |
| AuthZ | `FORBIDDEN` | 403 | 権限なし表示 | × | △ |
| NotFound | `NOT_FOUND` | 404 | Not Found / 空状態 | × | △ |
| Validation | `VALIDATION_FAILED` | 400/422 | フォームに反映 | × | × |
| Conflict | `CONFLICT` | 409 | 再読み込み/再試行案内 | △ | △ |
| RateLimit | `RATE_LIMITED` | 429 | 待って再試行（間隔表示） | ○ | ○ |
| Upstream | `UPSTREAM_TIMEOUT` | 504 | 再試行/フォールバック | ○ | ○ |
| Upstream | `UPSTREAM_UNAVAILABLE` | 502/503 | 再試行/フォールバック | ○ | ○ |
| Internal | `INTERNAL` | 500 | error boundary | × | ○ |
| Unknown | `UNKNOWN` | - | 汎用失敗 | △ | ○ |

---

## 実装方針（推奨）

- 例外はアプリ内の共通エラー型（例：`AppError`）へ正規化する
- 正規化は API 層（`shared/api`）に閉じ込め、features/pages に持ち込まない
- `requestId/traceId` をUIに表示するかは運用ポリシーで決める（サポート導線があるなら表示が有効）

---

## 禁止（AI/人間共通）

- `message.includes('...')` で分岐
- code の追加/変更をドキュメント無しで実装だけ入れる
