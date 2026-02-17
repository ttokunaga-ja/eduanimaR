# Quickstart（eduanima-librarian）

目的：Librarian の責務境界・契約・運用の「開始条件」を短時間で確定させる。

## 0) Phase別の開始条件（Must）

### Phase 1・Phase 2: Librarian未実装

**Phase 1・Phase 2ではLibrarianは実装しません。**

- Professorが直接Gemini 2.0 Flashを呼び出す
- Librarian推論ループは不要
- このドキュメントはPhase 3以降で参照する

### Phase 3: Librarian実装・統合開始

**Phase 3で初めてLibrarianを実装します。以下の条件を満たすこと:**

- [ ] サービス境界（Professor ↔ Librarian）が `01_architecture/MICROSERVICES_MAP.md` に反映されている
- [ ] Professor ↔ Librarian の契約（gRPC/Proto）が SSOT として確定している
  - SSOT: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
- [ ] CI の最低ゲート（lint/test/contract drift）が `05_operations/CI_CD.md` の方針で組める
- [ ] Professor側のgRPCクライアント実装が完了している
- [ ] Librarian未起動でもProfessorが動作する後方互換性が確保されている

## 1) 最短で読む順（推奨）

Phase 3開始時に以下を順に読んでください:

1. `01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`
2. `01_architecture/MICROSERVICES_MAP.md`
3. `01_architecture/CLEAN_ARCHITECTURE.md`
4. `02_tech_stack/STACK.md`（Phase別のLibrarian実装スケジュール参照）
5. `03_integration/API_CONTRACT_WORKFLOW.md`
6. `01_architecture/RESILIENCY.md`

## 2) まず埋める（プロジェクト固有）

Phase 3開始時に以下を確定させる:

- `00_quickstart/PROJECT_DECISIONS.md`
- LangGraph実装方針（検索ループ最大5回試行）
- Gemini 3 Flash のパラメータ設定（思考コスト・温度等）

## 3) Phase 3実装完了条件

以下をすべて満たすこと:

- [ ] Professor ↔ Librarian gRPC双方向ストリーミングが動作
- [ ] 検索ループが最大5回試行で停止
- [ ] Librarian未起動でもProfessorが動作（後方互換性）
- [ ] 検索精度がPhase 1/2より向上（検証質問で確認）

---

## 参照

- **Phase 1-5詳細**: `../../eduanimaRHandbook/04_product/ROADMAP.md`
- **Professor統合**: `../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`
- **gRPC契約**: `../../eduanimaR_Professor/proto/librarian/v1/librarian.proto`
