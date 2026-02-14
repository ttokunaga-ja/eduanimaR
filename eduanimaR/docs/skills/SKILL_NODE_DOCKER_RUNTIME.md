# SKILL: Node.js（Docker / Next standalone）

対象：本番相当の実行環境を揃え、環境差分事故を減らす。

変化に敏感な領域：
- Node version 差
- Docker base image（alpine等）差

関連：
- `../05_operations/RELEASE.md`
- `../05_operations/CI_CD.md`

---

## Versions（2026-02-11 / official）

Node（公式 index.json）：
- latest LTS：`v24.13.1`（Krypton）
- latest Current：`v25.6.1`

運用の推奨：
- 本番/CI は LTS を固定し、Current は検証用途に留める

---

## Must
- Node version を固定する（CI/本番）
- `next build` を必須にし、standalone 前提の動作を確認する

## 禁止
- ローカルだけ動く前提の導入（ネイティブ依存の取り扱い無計画）

## チェックリスト
- [ ] CIと本番の Node が揃っているか？
- [ ] production build で動くか？
