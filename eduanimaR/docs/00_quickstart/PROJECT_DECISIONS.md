# Project Decisions（SSOT）

このファイルは「プロジェクトごとに選択が必要」な決定事項の SSOT。
AI/人間が推測で埋めないために、まずここを埋めてから実装する。

## 基本
- プロジェクト名：
- リポジトリ：
- 対象環境：local / staging / production

## 認証（Must）
- 方式：Cookie / Bearer（どちらかに統一）
- セッション保存場所：Cookie / memory / storage（Bearer の場合は特に）
- 401/403 の UI 振る舞い：

## API（Must）
- OpenAPI の取得元：
- OpenAPI の配置パス（このrepo内）：`openapi/openapi.yaml` / 変更（理由：）
- 生成物の配置：`src/shared/api/generated`（固定）

## Next.js（Must）
- SSR/Hydration：原則 Must（例外ページがあれば列挙）
- Route Handler/Server Action の採用方針：
- キャッシュ戦略（tag/path/revalidate の主軸）：

## i18n（採用時 Must）
- 対象言語：
- 翻訳ファイルの置き場：
- 直書き文字列の扱い（lint/CI）：

## 観測性（Must）
- エラー通知：
- Web Vitals / RUM：
- ログの取り扱い（PII/Secrets）：
