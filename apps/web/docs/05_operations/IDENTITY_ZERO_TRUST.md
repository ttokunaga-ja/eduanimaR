# Identity / Zero Trust（Frontend / BFF）

このドキュメントは、フロントエンド（Browser + Next BFF）での認証/認可の前提を固定します。

関連：
- スタック：`../02_tech_stack/STACK.md`
- セキュリティヘッダー/CSP：`../03_integration/SECURITY_CSP.md`

---

## 結論（Must）

- ブラウザから microservices を直接叩かない（入口はBFF/Go Gatewayに寄せる）
- トークンは “保存しない” を基本（必要ならHTTP-only Cookie等、攻撃面を最小化）
- 認可は server 側で最終判定（クライアントの状態を信用しない）

---

## 1) 認証方式（テンプレ）

プロジェクトで確定する：
- Cookie（推奨：同一オリジン/BFF）
- Bearer（どうしても必要な場合）

---

## 2) CSRF / XSS の前提

- Cookie を使うなら CSRF 対策が必要（same-site + トークン等）
- XSS を起こすと認証が破られるため、CSP/入力/サニタイズを軽視しない

---

## 3) ログイン状態の扱い

- “ログイン済み” は server 側で確定させる
- UIは server の結果に追随し、勝手に権限を決めない

---

## 4) 監査（任意）

- 重要操作（設定変更/権限変更）は監査ログと整合させる
- フロントは requestId/traceId を引き回せると切り分けが容易

---

## 禁止（AI/人間共通）

- localStorage にアクセストークンを保存
- クライアント側の条件分岐だけで権限を担保
- “とりあえず CORS で直叩き”
