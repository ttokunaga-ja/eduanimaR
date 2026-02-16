# Observability（ログ/エラー/計測）

Last-updated: 2026-02-16

このドキュメントは、運用フェーズでの「見えない」をなくすために、
ログ・エラー・パフォーマンス計測の最小契約を固定します。

---

## 結論（Must）

- 例外/障害は “検知できる形” で残す（握りつぶさない）
- ブラウザ体験（Web Vitals）とサーバ側の失敗（5xx/タイムアウト）を分けて観測する
- Next.js の境界（RSC/Route Handler/Client）ごとにログ責務を分ける

---

## SLO/アラート（Must）

観測は「測る」だけで終わらせず、**運用の判断**（ページャ/チケット/改善）に繋げる必要があります。

- SLO とアラートの最小セット：`SLO_ALERTING.md`
- リリース相関：ログ/メトリクスに `releaseId`（ビルド番号/コミットSHA等）を付与し、リリース前後比較を可能にする
- 取得元の分離：
  - RUM（ブラウザ）：Core Web Vitals / JS エラー率 / 画面表示成功率
  - BFF（Route Handler / Server Action）：5xx 率 / upstream 失敗 / latency

---

## ログ

- 形式：構造化（JSON）を推奨
- 必須フィールド（最低限）：
  - `requestId`（または trace id）
  - `route` / `method`
  - `userId`（PII にならない識別子）
  - `status` / `latencyMs`
- 禁止：アクセストークン/セッション情報など secrets の出力

---

## 追加：実装契約（Must）

### requestId / traceId

- requestId（または trace id）は **サーバ境界（Route Handler / Server Action）で生成 or 受け取り**、ログに必ず載せる
- UI に出す場合は、サポート導線があるときのみ（PII/攻撃面を増やさない）

### どこで何を観測するか

- RSC：初期表示に影響する失敗・遅延（ページ単位）
- Route Handler / Server Action：HTTP 境界の status / latency / upstream 失敗
- Client：ユーザー操作起点の失敗（mutation）と Web Vitals

### PII/Secrets Redaction（禁止の具体化）

- Authorization ヘッダー、Cookie、セッショントークン、API key はログに出さない
- エラー出力に request body を丸ごと含めない（必要ならフィールド単位で許可リスト）

## OpenTelemetryとの統合（推奨）

### trace/log correlationを前提とした設計

- **request_id/trace_idでログ相関**: フロントエンド → Professor → Librarianの全ログにrequest_id/trace_idを含める
- **分散トレーシング**: OpenTelemetryを使用して、リクエストの全体像を追跡

### Professor APIとのトレース連携

フロントエンド → Professor → Librarianのトレースを一貫して追跡:

```typescript
// トレースID伝搬の例
const response = await fetch('/api/professor/qa/ask', {
  headers: {
    'X-Request-ID': requestId,
    'traceparent': traceParent, // W3C Trace Context
  },
})
```

### SLO/アラート（運用基準の適用）

バックエンドと同様の運用基準を適用:

- **Core Web Vitals**: LCP/INP/CLS の目標値設定
- **エラー率**: 5xx エラー率の監視
- **レイテンシ**: P95/P99 レイテンシの監視

**参照元SSOT**:
- `../../eduanimaR_Professor/docs/05_operations/OBSERVABILITY.md`
- `../../eduanimaR_Librarian/docs/02_tech_stack/STACK.md` (Observability)

---

## エラー

- UI：Next の error boundary（route error / global error）で “ユーザーに見せる失敗” を統一
- サーバ：Route Handler / Server Action では、エラー分類（4xx/5xx）を揃える

---

## Web Vitals（RUM）

- 最小：Core Web Vitals を収集し、リリース前後で比較できる状態にする
- 目的：SSR/Hydration を採用する以上、LCP/INP/CLS の悪化を検知できるようにする

---

## トレーシング（任意）

- 重要 API のみでも良いので、分散トレーシングを導入する（OpenTelemetry 等）
- “Next（BFF）→ Go Gateway → Service” の遅延分解ができる状態が理想
