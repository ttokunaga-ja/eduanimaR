# Performance / Load Testing（Frontend）

このドキュメントは、フロントエンド（Next.js BFF + Browser）における性能検証を
“気合い”ではなく、最低限の手順と指標として固定します。

関連：
- 運用の性能：`../05_operations/PERFORMANCE.md`
- 観測性：`../05_operations/OBSERVABILITY.md`

---

## 結論（Must）

- リリース前後で **Core Web Vitals** と **エラー率** を比較できる状態にする
- “速い/遅い” は体感ではなく指標で判断する
- 性能検証は「どのページがSLO対象か」を先に決める

---

## 1) 対象の決め方（テンプレ）

- 重要導線（例：ログイン後トップ、検索、購入、設定）
- SEO/流入が大きいページ
- 失敗するとビジネス影響が大きいページ

---

## 2) 収集する指標（最小）

### ブラウザ（RUM）
- LCP / INP / CLS（Core Web Vitals）
- ページ別のエラー率（JSエラー、API失敗）

### サーバ（Next.js / BFF）
- ルート別：p50/p95/p99 latency
- 5xx / timeout
- upstream（Go Gateway）呼び出しの latency

---

## 3) 検証の観点

- SSR/Hydration：初期表示が白画面になっていないか
- Cache：意図せず動的化していないか（Dynamic API の利用箇所）
- API 呼び出し：N+1 や重複呼び出しがないか
- Client 化：不要な `use client` が増えていないか

---

## 4) 合成監視（任意だが推奨）

- 主要導線を headless で定期実行し、
  - 失敗率
  - 所要時間
  - 直近リリースとの比較
  を見る

---

## 5) 負荷（Load）テストの考え方（BFF）

フロントの“負荷”は、次の2種類に分ける：
- ブラウザ体験（RUMで実測）
- BFF（Next）/upstream（Go）への同時アクセス耐性

最低限：
- 主要ページ（サーバレンダー）の同時アクセスで p95 が崩れない
- upstream が遅い/落ちる時のフォールバックが成立する（`RESILIENCY.md`）

---

## 禁止（AI/人間共通）

- 目標値なしで計測する（比較できない）
- “遅いから no-store” で逃げる（コスト増/根本未解決）
- 本番だけで初めて性能問題に気づく
