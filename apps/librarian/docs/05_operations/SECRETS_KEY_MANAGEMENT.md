# SECRETS_KEY_MANAGEMENT

## 目的
秘密情報（secrets）と暗号鍵（keys）を安全に扱い、漏えい時の影響を最小化する。

## 適用範囲
- アプリ設定（DB接続、外部APIキー、イベント基盤の認証、TLS鍵など）
- 署名鍵（JWT/セッショントークン等）
- 暗号鍵（PIIの暗号化、KMS、Envelope encryption）

## 分類
### Secret（秘密情報）
- DB パスワード / 接続情報
- 外部 API キー
- OAuth client secret
- Webhook secret

### Key（鍵）
- 署名鍵（JWT signing key 等）
- 暗号鍵（データ暗号化、フィールド暗号化）
- mTLS 証明書の private key

### 非Secret（設定）
- feature flag の ON/OFF
- タイムアウト値
- ログレベル

## 原則（MUST）
- **秘密情報を Git にコミットしない**（例外なし）
- **最小権限**: secret は “必要なワークロードのみ” が読める
- **ローテーション前提**: 期限/ローテ計画を持つ
- **平文で保存しない**: KMS/Secret Manager 等で暗号化保管
- **ログに出さない**: 例外メッセージ・設定ダンプにも出さない

## 保管（推奨）
- 組織標準の Secret Manager を正とする
- 可能なら KMS による暗号化 + 監査ログ（誰が読んだか）を有効化

## 注入（Injection）パターン
### 1) 環境変数（MUST）
- コンテナ起動時に secret manager から注入
- アプリは `os.Getenv` 等で読む

### 2) ファイルマウント（SHOULD）
- TLS 証明書など、ファイルの方が扱いやすいものに向く
- パーミッションは最小にする

### 禁止（SHOULD NOT）
- イメージに埋め込む
- `.env` をリポジトリに置く（ローカル専用で `.gitignore` されている場合は可）

## ローテーション
### ローテ対象（最低限）
- 外部 API キー
- DB ユーザー（可能ならアプリ専用）
- 署名鍵（JWT 等）

### 方式（推奨）
- **二重運用（dual key）**:
  - 署名鍵は「旧鍵で検証、新鍵で署名」を一定期間併用
  - 期限後に旧鍵を廃止
- DB パスワードは “新パスワード発行 → 並行稼働 → 切替 → 旧廃止”

## 漏えい時の対応（MUST）
- 1) 影響範囲を特定（どの secret / key か）
- 2) 失効・ローテーション（可能なら自動）
- 3) 監査ログ確認（いつ誰がアクセスしたか）
- 4) 二次被害の封じ込め（レート制限、IP制限、権限削減）
- 5) ポストモーテム（関連: `05_operations/INCIDENT_POSTMORTEM.md`）

## 署名鍵（JWT等）の追加ルール
- 鍵ID（`kid`）を使い、ローテーションを前提にする
- 署名アルゴリズムは allowlist（`alg=none` 等を拒否）

## 暗号鍵（データ暗号化）
- **Envelope encryption** を推奨:
  - データキー（DEK）でデータを暗号化
  - DEK を KMS のキー暗号鍵（KEK）でラップ
- キーの用途（purpose）を分離（署名と暗号で鍵を共用しない）

## CI/CD 連携（MUST）
- secret scanning（コミット/PR）を有効化
- CI の secret は最小権限 + 短命（OIDC 等）を優先

## 関連
- `05_operations/SUPPLY_CHAIN_SECURITY.md`
- `05_operations/CI_CD.md`
- `05_operations/IDENTITY_ZERO_TRUST.md`
