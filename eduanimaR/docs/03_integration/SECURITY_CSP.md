# Security Headers & CSP Policy（Next.js App Router）

このドキュメントは、ブラウザ向けセキュリティ（CSP/各種ヘッダー）を **プロジェクトの契約** として固定し、
- XSS
- クリックジャッキング
- 意図しない外部通信
のリスクを下げます。

---

## 結論（Must）

- セキュリティヘッダーは Next.js の設定で一元管理する
- CSP は “何を守るか” と “性能/運用コスト” のトレードオフを明文化して選ぶ

参考（Next.js ガイド）：
- Content Security Policy（CSP）は **nonce 方式** と **non-nonce（静的ヘッダー）方式** で運用が変わります
- nonce 方式は **動的レンダリング必須** になり、CDN キャッシュ等と相性が悪くなる

---

## 採用方針（プロジェクトで確定させる）

以下から 1 つを選び、ここに確定値を記載してください。

**本テンプレート（確定）：Option B（nonce-based / Strict CSP）を採用する。**

### Option A: non-nonce CSP（まずはこれ）
- next.config の `headers()` で CSP を配布
- 運用コストが低い
- ただし厳密にやると `unsafe-inline` 等の調整が必要

### Option B: nonce CSP（高セキュリティ）
- `proxy.ts` 等で nonce を生成し、リクエスト毎に CSP を付与
- セキュリティ強度は上げやすい
- 代償：ページが動的化しやすい / キャッシュとの相性 / 運用複雑性

### Option C: SRI（実験機能）
- 静的生成を保ちつつ strict CSP に近づけられる
- ただし実験的で、バンドラ等の制約がある

---

## 最小セットのセキュリティヘッダー（推奨）

ここに “プロジェクトとして採用するヘッダー” を列挙します（テンプレ）。

- `Strict-Transport-Security`（HSTS。https 前提の本番で）
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy`
- `Permissions-Policy`（必要機能のみ許可）
- `Content-Security-Policy`（CSP）
- `frame-ancestors`（CSP 側で clickjacking 防止）

注意：`X-Frame-Options` は legacy のため、基本は CSP の `frame-ancestors` を優先。

### Librarian呼び出し禁止の明記（CSPレベル）

**CSPヘッダーでLibrarianへの直接通信をブロック**:

```typescript
// proxy.ts または next.config でのCSP設定例
const csp = [
  "default-src 'self'",
  `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'`,
  `style-src 'self' 'nonce-${nonce}'`,
  "connect-src 'self' https://professor.example.com", // Professorのみ許可
  // Librarianへの直接通信は許可しない
  "img-src 'self' blob: data:",
  "font-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
  'upgrade-insecure-requests',
].join('; ')
```

**許可されるAPI**:
- Professor API: `https://professor.example.com` (本番環境)
- Professor API: `http://localhost:8080` (ローカル開発)

**Chrome拡張機能の特殊対応**:
- `manifest.json` の `content_security_policy` との整合性を保つ
- 拡張機能からのProfessor API通信も同様の制約を適用

**参照元SSOT**:
- `../../eduanimaR_Professor/docs/02_tech_stack/TS_GUIDE.md` (Librarian呼び出し禁止)

---

## 実装場所（推奨）

- **non-nonce**：`next.config.*` の `headers()`
- **nonce**：`proxy.ts`（Next.js の Proxy file convention）で nonce を生成し `Content-Security-Policy` と `x-nonce` を付与

---

## nonce-based CSP 実装テンプレ（Must）

### 1) `src/app/proxy.ts` を追加する

- リクエスト毎に nonce を生成し、`Content-Security-Policy` に埋め込む
- Next.js が自動で nonce を適用できるよう、`x-nonce` を付与する
- `next/link` の prefetch と静的アセットには不要なので matcher で除外する

実装例（テンプレ）：

```ts
import { NextRequest, NextResponse } from 'next/server'

export function proxy(request: NextRequest) {
	const nonce = Buffer.from(crypto.randomUUID()).toString('base64')
	const isDev = process.env.NODE_ENV === 'development'

	const csp = [
		"default-src 'self'",
		`script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${isDev ? " 'unsafe-eval'" : ''}`,
		`style-src 'self' 'nonce-${nonce}'${isDev ? " 'unsafe-inline'" : ''}`,
		"img-src 'self' blob: data:",
		"font-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		'upgrade-insecure-requests',
	].join('; ')

	const requestHeaders = new Headers(request.headers)
	requestHeaders.set('x-nonce', nonce)
	requestHeaders.set('Content-Security-Policy', csp)

	const response = NextResponse.next({
		request: { headers: requestHeaders },
	})
	response.headers.set('Content-Security-Policy', csp)
	return response
}

export const config = {
	matcher: [
		{
			source: '/((?!api|_next/static|_next/image|favicon.ico).*)',
			missing: [
				{ type: 'header', key: 'next-router-prefetch' },
				{ type: 'header', key: 'purpose', value: 'prefetch' },
			],
		},
	],
}
```

注意：
- `script-src` / `connect-src` などはプロジェクトが利用する外部ドメインに合わせて拡張する
- 例：Google Tag Manager 等を入れるなら、該当ドメインを許可し、`nonce` を渡す

### 2) dynamic rendering の影響を理解する（Must）

nonce は「リクエストごとに異なる値」です。
そのため、nonce-based CSP を正しく運用するには **動的レンダリングが必要** になり、
静的最適化/ISR/CDN キャッシュとのトレードオフが発生します。

本番での影響が大きい場合は、ページ単位で設計判断（どのページを動的にするか）を明文化してください。

### 3) nonce の読み出し（必要な場合）

第三者スクリプト等で `<Script nonce={...} />` が必要なら、Server Component で `x-nonce` を読む：

```ts
import { headers } from 'next/headers'

export async function getNonce() {
	return (await headers()).get('x-nonce')
}
```

---

## セキュリティヘッダーの一元管理（補足）

- CSP は `proxy.ts` で動的に付与する
- それ以外（HSTS / nosniff / Referrer-Policy / Permissions-Policy など）は `next.config.*` の `headers()` で静的に付与してよい

---

## 運用チェック（最低限）

- 本番で主要ページが “意図せず静的化/動的化” していないか（キャッシュ/費用/速度に直結）
- CSP 違反レポート（必要なら `report-to` / `report-uri`）の扱いを決める
- 追加する外部ドメインは最小化し、レビュー対象にする

---

## データセキュリティ（RSC 時代の注意）

- Server Components は secrets/DB へアクセスできるが、Client Components は “ブラウザコードと同等” の前提で扱う
- Client に渡すデータは DTO 最小化（DAL で制御）

関連： [../01_architecture/DATA_ACCESS_LAYER.md](../01_architecture/DATA_ACCESS_LAYER.md)
