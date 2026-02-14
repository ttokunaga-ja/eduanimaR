# Supply Chain Security（Frontend）

このドキュメントは、フロントエンドのビルド/依存関係に関するサプライチェーン対策を
最低限の運用として固定します。

---

## 結論（Must）

- 依存の入力（package/lockfile）を固定し、再現可能なビルドにする
- CI で“どこから来たビルドか”を追える状態にする
- 生成物（API client 等）を手編集しない（SSOTを守る）

---

## 1) 再現性

- `npm ci` を基本（lockfile準拠）
- ビルド環境（Node version）を固定する

---

## 2) 依存の最小化

- 使っていない依存は削除
- 似た機能を複数入れない（攻撃面が増える）

---

## 3) 来歴（Provenance）

- リリース成果物に、ビルド情報（commit/tag、CI run）を紐づける
- SBOM を作る/保存する運用を検討する（プロジェクト要件に応じて）

---

## 4) 生成物の取り扱い

- Orval 生成物は CI で差分検知
- OpenAPI をSSOTとして扱う

関連：`../03_integration/API_CONTRACT_WORKFLOW.md`

---

## 禁止（AI/人間共通）

- 依存を“とりあえず追加”する
- 生成物の手編集
- 秘密情報をビルドに埋め込む
