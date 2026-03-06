# SKILL: TypeScript（型安全・生成物優先）

対象：Go⇔TS の型同期（OpenAPI生成）を前提に、型を “契約の道具” として使う。

変化に敏感な領域：
- TS の厳格化オプション
- 生成型をそのままUI propsに流す事故

関連：
- `../03_integration/API_CONTRACT_WORKFLOW.md`
- `../01_architecture/DATA_ACCESS_LAYER.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `typescript`: `5.9.3`
- `@types/react`: `19.2.13`
- `@types/react-dom`: `19.2.3`

---

## Must
- API 由来の型は生成物をSSOTにする
- UI props は DTO 最小化（生成型の丸渡し禁止）

### 実装メモ（テンプレの前提）

- `tsconfig.json` は `moduleResolution: Bundler` を前提にしている
- alias は `@/*` → `src/*`

## 禁止
- `any` で逃げる
- 生成型をそのまま Client props にする

## チェックリスト
- [ ] 生成型と UI props が分離されているか？
- [ ] 不要なフィールドをクライアントに渡していないか？
