# Quickstart（eduanima-librarian）

目的：Librarian の責務境界・契約・運用の「開始条件」を短時間で確定させる。

## 0) Phase 1開始条件（Must）

**Phase 1でLibrarianのすべての機能を完全に実装します。**

以下の条件を満たすことで、Phase 1の開発を開始できます:

- [ ] サービス境界（Professor ↔ Librarian）が `01_architecture/MICROSERVICES_MAP.md` に反映されている
- [ ] Professor ↔ Librarian の契約（gRPC/Proto）が SSOT として確定している
  - SSOT: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
- [ ] CI の最低ゲート（lint/test/contract drift）が `05_operations/CI_CD.md` の方針で組める
- [ ] Professor側のgRPCクライアント実装準備が整っている
- [ ] Librarian側のgRPCサーバー実装準備が整っている

## 1) 最短で読む順（推奨）

Phase 1開始時に以下を順に読んでください:

1. `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
2. `01_architecture/MICROSERVICES_MAP.md`
3. `01_architecture/CLEAN_ARCHITECTURE.md`
4. `02_tech_stack/STACK.md`（Phase別のLibrarian実装スケジュール参照）
5. `03_integration/API_CONTRACT_WORKFLOW.md`
6. `01_architecture/RESILIENCY.md`

## 2) まず埋める（プロジェクト固有）

Phase 1開始時に以下を確定させる:

- `00_quickstart/PROJECT_DECISIONS.md`
- LangGraph実装方針（検索ループ最大5回試行）
- Gemini 3 Flash のパラメータ設定（思考コスト・温度等）

## 3) Phase 1実装完了条件

以下をすべて満たすこと:

- [ ] Professor ↔ Librarian gRPC双方向ストリーミングが動作
- [ ] 検索ループが最大5回試行で停止
- [ ] LangGraphで状態管理が正常動作
- [ ] Gemini 3 Flashでの推論が動作（Plan/Evaluate）
- [ ] エビデンス選定が正常動作
- [ ] 検索成功率70%以上（10件の検証質問）
- [ ] 検索応答時間p95で5秒以内
- [ ] ハルシネーション率20%以下

---

## 参照

- **Phase 1-5詳細**: `../../eduanimaRHandbook/04_product/ROADMAP.md`
- **Professor統合**: `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`
- **gRPC契約**: `../../eduanimaR_Professor/proto/librarian/v1/librarian.proto`
