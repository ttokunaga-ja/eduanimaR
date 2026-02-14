# Incident / Postmortem（Frontend）

このドキュメントは、フロントエンド起因の障害に対する
インシデント対応とポストモーテムの最小テンプレを固定します。

---

## 結論（Must）

- “誰が悪いか” ではなく “再発を防ぐ契約” を作る
- 事実（時系列）と判断（なぜそうしたか）を分けて書く
- 恒久対応は docs を更新し、次に同じ判断をしない

---

## 1) 典型的なフロント事故（例）

- CSP nonce による意図せぬ動的化 → コスト/速度悪化
- 環境変数の誤設定 → API baseURL が本番だけ違う
- 生成物ズレ（OpenAPI/Orval） → 実行時エラー
- キャッシュ誤爆（tag/path） → 古いデータが残る
- SSR/Hydration 崩れ → 初期表示が白画面
- feature flag の切替ミス

---

## 2) 初動（テンプレ）

- 影響範囲（ページ/ユーザー/地域）
- 重大度（SLO逸脱、決済影響など）
- 直近リリース有無
- 迂回策（ロールバック/機能OFF）

---

## 3) 切り分け（最小）

順序：
1. RUM（Vitals/JSエラー）
2. Next（5xx/latency）
3. upstream（Go Gateway 以降）

関連：`OBSERVABILITY.md`

---

## 4) ポストモーテム（テンプレ）

- Summary（何が起きたか）
- Impact（誰にどう影響したか）
- Timeline（時系列）
- Root cause（技術的原因）
- Contributing factors（運用/手順/監視の穴）
- Detection（どう検知したか、なぜ遅れたか）
- Resolution（どう直したか）
- Action items（再発防止：Owner/期限/検証方法）
- Docs updates（更新した契約）

---

## 禁止（AI/人間共通）

- 原因を“人”に帰属させる
- 再発防止がコードだけで、契約（docs/CI/監視）が変わらない
