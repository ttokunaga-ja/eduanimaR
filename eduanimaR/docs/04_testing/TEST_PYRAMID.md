# Test Pyramid（Frontend）

このドキュメントは、フロントエンド（Next.js + FSD）におけるテスト配分（ピラミッド）を固定し、
- E2E 過多
- fragile なスナップショット大会
- 境界違反（テストが deep import の温床）
を防ぐための契約です。

関連：
- 戦略：`TEST_STRATEGY.md`

---

## 結論（Must）

- “速いテスト” を土台にし、E2E は最小にする
- 重要ユースケースは **Component/Integration** で厚くする
- テストも FSD の Public API 経由（deep import 禁止）

---

## 推奨配分（目安）

- Unit：多（純粋関数/変換/バリデーション）
- Component：多（UIの主要状態・操作）
- Integration：中（feature + API hook の結合）
- E2E：少（主要導線の“煙検知”）

重要：割合ではなく「壊れた時の影響が大きいところ」に投資する。

---

## レイヤー別のテスト責務（目安）

- `shared/lib`：Unit
- `shared/ui`：Component
- `entities/*/ui`：Component
- `features/*/ui`：Component / Integration
- `widgets`：必要なら Component（結合が増えやすいので最小）
- `pages`：原則テストしない（ページは合成なので、下で担保する）

---

## E2E を最小にする理由（実務）

- 不安定になりやすい（環境/タイミング/外部依存）
- デバッグコストが高い
- 失敗時に原因特定が遅い

E2E の価値は「統合の煙」を早期に出すこと。
UIの細部を E2E で担保しない。

---

## 最低限の E2E（テンプレ）

- ログイン（またはゲスト導線）
- 主要ページ表示（初期表示のSSR/Hydration崩れ検知）
- 重要な mutation（1つ）

---

## 禁止（AI/人間共通）

- 重要ロジックを E2E だけで担保する
- スナップショットで UI 全体を固定する（変更耐性が低い）
- テストのために境界設定（boundaries）を緩める
