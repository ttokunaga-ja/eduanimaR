# Secrets / Key Management（Frontend）

このドキュメントは、Next.js（RSC/BFF）における secrets/keys の扱いを固定し、
漏洩事故を防ぐための契約です。

---

## 結論（Must）

- `NEXT_PUBLIC_` は “ブラウザ公開” と同義（秘密は絶対に置かない）
- secrets は server-only（RSC / Route Handler / Server Action）に閉じ込める
- 環境差分は `.env` とドキュメントに集約し、コードにハードコードしない

---

## 1) 環境変数の分類

- Public：`NEXT_PUBLIC_*`（クライアントへバンドルされる）
- Private：`*`（server-onlyで参照する）

Must：
- Private 変数は client から import されるモジュールで参照しない

---

## 2) 管理方針（テンプレ）

- ローカル：`.env.local`（コミットしない）
- staging/production：シークレットストアで注入

運用：
- ローテーション手順
- 失効時の影響範囲

---

## 3) ログへの出力禁止

- アクセストークン、セッション、APIキー、Cookie値
- Authorization header

関連：`OBSERVABILITY.md`

---

## 4) 第三者連携（Analytics等）

- Public key しか要らない形にする
- もし secret が必要なら server 経由にする（ブラウザへ出さない）

---

## 禁止（AI/人間共通）

- クライアントコードへ secret を渡す
- `.env` をリポジトリへコミットする
- 例外を増やして “隠す” （正しい境界へ移す）
