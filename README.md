# codingAgent_docs

このリポジトリは、AI（Coding Agent）と人間が「迷わずに」開発を開始できるように、
設計・運用・契約（SSOT）をテンプレート化したドキュメント/雛形集です。

## Templates

- Frontend / FSD（Next.js + Feature-Sliced Design）
	- [codingAgent_FeatureSlicedDesign_template/README.md](codingAgent_FeatureSlicedDesign_template/README.md)

- Backend / Microservices（Go + Clean Architecture + 契約駆動 + 運用）
	- [codingAgent_MicroServicesArchitecture_template/README.md](codingAgent_MicroServicesArchitecture_template/README.md)

## 使い方（最短）

1. 目的に近いテンプレの `.cursorrules` と `docs/` をプロジェクトへ取り込む
2. 各テンプレの `docs/README.md`（Docs Portal）に従って読む順を揃える
3. 迷ったら `docs/skills/`（短い実務ルール）を先に確認する

## 基本方針

- “実装より先にドキュメント（契約）を更新する”
- SSOT（Single Source of Truth）を明示し、推測での実装を避ける
- 例外を増やさず、境界/責務/契約を直して解決する
