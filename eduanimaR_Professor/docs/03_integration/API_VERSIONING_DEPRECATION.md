# API_VERSIONING_DEPRECATION

## 目的
外部API（OpenAPI）を長期運用するため、互換性・バージョニング・非推奨/廃止（deprecation）を標準化する。

## 適用範囲
- Browser ↔ Next.js(BFF)
- Next.js(BFF) ↔ Go API Gateway（外向き API / OpenAPI SSOT）

## 原則（MUST）
- **互換性を壊さない**のが基本。破壊的変更は最後の手段。
- “互換性の判断”はレビューで曖昧にせず、ルール化する。
- 非推奨（deprecate）と廃止（remove）の手順・期限を必ず決める。

## バージョニング方式
### 推奨: パスバージョニング
- 例: `/v1/...`
- 利点: 明確で運用しやすい

### 代替: ヘッダ/メディアタイプ
- 例: `Accept: application/vnd.example.v1+json`
- 組織のAPI運用が成熟している場合に検討

> どの方式でも「廃止手順」と「契約テスト」が揃っていることが重要。

## 互換性ルール（OpenAPI）
### 互換（Backward compatible）として扱える変更（例）
- 新しいエンドポイント追加
- response に**任意**フィールド追加
- request に**任意**フィールド追加（サーバ側で無視可能）
- enum に新しい値を追加（ただしクライアントが unknown を安全に扱える設計が必要）

### 破壊的（Breaking）変更（例）
- 필수（required）を増やす
- フィールド削除/型変更/意味変更
- enum の値削除/意味変更
- エラーコード（`error.code`）の再定義（互換性破壊になりやすい）

## Deprecation（非推奨）
### MUST
- 非推奨にする対象と理由を記録
- 置き換え手段（migration path）を提示
- 期限（sunset date）を決める

### 推奨ヘッダ（例）
- `Deprecation: true`
- `Sunset: <date>`
- `Link: <replacement>; rel="successor-version"`

> 実装可否はプロジェクトで決める。重要なのは “期限と移行手順” を明文化すること。

## 廃止（Removal）
- 1) 事前告知（ドキュメント・リリースノート）
- 2) 監視で利用者を把握（どのクライアントが呼んでいるか）
- 3) 期限到来後に削除
- 4) 破壊が起きた場合の問い合わせ導線を用意

## API インベントリ（MUST）
- OpenAPI を SSOT として維持
- “公開中/非推奨/廃止予定” を追跡できる状態にする

## 関連
- `03_integration/API_CONTRACT_WORKFLOW.md`
- `03_integration/CONTRACT_TESTING.md`
- `05_operations/API_SECURITY.md`
