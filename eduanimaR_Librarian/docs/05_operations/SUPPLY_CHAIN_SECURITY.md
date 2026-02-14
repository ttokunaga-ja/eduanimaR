# SUPPLY_CHAIN_SECURITY

## 目的
依存関係・ビルド・配布の経路を保護し、改ざんや混入リスクを下げる。
SLSA の考え方（共通言語）を、プロジェクトの CI/CD と運用に落とす。

> 注: SLSA は継続更新されるため、採用時は現行版（例: v1.2）を参照する。

## 最小要件（MUST）
- 依存関係の脆弱性検査を CI に含める（Go / Node / コンテナ）
- 生成物（コンテナイメージ）を **再現可能な手順** でビルドし、出自を追える状態にする
- 生成ツール/コード生成のバージョンを固定し、再生成差分を CI で検出する（既存方針を強化）

## 推奨（SHOULD）
- SBOM を生成し、リリース成果物に紐づける
- ビルド provenance（少なくとも「どのソース/どのCIで作られたか」）を保存
- 重要成果物に署名（イメージ署名など）を検討

## “作る”だけでなく“検証する”（推奨）
供給網対策は、
- provenance/署名を生成し
- 配布し
- **利用側（CD/実行環境）で検証して拒否できる**
ところまで到達して初めて効果が出る。

### 最小の検証ゲート（SHOULD）
- “署名済み” の成果物のみデプロイ可
- provenance が存在し、期待するリポジトリ/ワークフロー由来であること
- SBOM が成果物に紐づいていること

## チェックリスト
- Dependencies
  - Go: `go mod` の変更は PR でレビュー
  - Node: lockfile を必須（pnpm-lock/yarn.lock/package-lock 等）
- CI
  - 依存スキャン、コンテナスキャン、SBOM生成がパイプラインに存在
  - 最小権限（CI token/secret のスコープ最小化）
- Release
  - リリースタグ/ビルド番号と成果物がトレース可能

## 関連
- 05_operations/CI_CD.md
- 05_operations/RELEASE_DEPLOY.md
- 05_operations/VULNERABILITY_MANAGEMENT.md
