# SKILL: Zod + React Hook Form（Forms / Validation）

対象：入力→検証→送信→失敗表示を一貫させ、仕様漏れを減らす。

変化に敏感な領域：
- Validation の置き場が散る（features外へ漏れる）
- APIの validation error とフォーム表示の不整合

関連：
- `../03_integration/ERROR_HANDLING.md`
- `../03_integration/ERROR_CODES.md`
- `../01_architecture/FSD_LAYERS.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `zod`: `4.3.6`
- `react-hook-form`: `7.71.1`

---

## Must
- フォームは `features` に置く（ユースケース責務）
- Zod スキーマをSSOTにし、UI/送信/エラー表示を揃える

## 禁止
- validation ロジックを pages/widgets に散らす
- APIのエラー文字列を直接UIに出す（code→UIへ正規化）
- UIに表示する全ての文言は翻訳キー（変数）で管理し、翻訳JSONから読み出すこと。フォームのエラー表示も翻訳キーを使用し、未翻訳キーは CI で検出する

## チェックリスト
- [ ] 成功/失敗の分岐がUIとして定義されているか？
- [ ] validation と API error の表示が矛盾していないか？
