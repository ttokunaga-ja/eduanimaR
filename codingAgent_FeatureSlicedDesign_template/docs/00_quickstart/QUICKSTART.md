# Quickstart（Frontend / FSD Template）

目的：このテンプレを取り込んだ直後に、迷わず「起動できる / 生成できる / CIゲートが作れる」状態にする。

## 0) 前提
- Node.js：LTS 推奨
- パッケージマネージャ：このテンプレでは `npm` 表記に統一

## 1) ローカル起動（最短）
```bash
npm install
npm run dev
```

## 2) 環境変数（Must）
- `.env.example` をベースに `.env.local` を作成し、baseURL を確定する
  - `NEXT_PUBLIC_API_BASE_URL`
  - `API_BASE_URL`

SSOT：`03_integration/DOCKER_ENV.md` と `05_operations/RELEASE.md`

## 3) API 生成（契約駆動の入口）
1. OpenAPI を `openapi/openapi.yaml` に配置（または `orval.config.ts` を変更）
2. 生成：
```bash
npm run api:generate
```

SSOT：`03_integration/API_GEN.md`

## 4) 最低限の品質ゲート（CIの骨格）
ローカルで以下が通る状態を「開始条件」とする：
```bash
npm run lint
npm run typecheck
npm test
npm run build
```

SSOT：`05_operations/CI_CD.md`

## 5) 次に埋める（プロジェクト固有）
- `00_quickstart/PROJECT_DECISIONS.md`
- `03_integration/DOCKER_ENV.md`
- `03_integration/API_VERSIONING_DEPRECATION.md`
