# Docker Environment & Microservices Integration

このドキュメントは、フロントエンドがマイクロサービス群と連携するための「実行方法」「接続先」「注意点」を固定し、AI が勝手な URL / ポート / 認証方式を想定しないための契約です。

## 前提
- 開発時のバックエンド起動方式：`docker compose` を想定
- フロントエンドの起動方式：Next.js（App Router）

## 接続先（プロジェクト固有で必ず埋める）
- API Base URL (local)：例 `http://localhost:8080`（※実際の値をここに確定させる）
- API Base URL (staging)：例 `https://api-stg.example.com`
- 認証方式：`Cookie` or `Bearer`（どちらかに統一する）

環境変数（推奨）：
- `NEXT_PUBLIC_API_BASE_URL`：ブラウザから参照する API base
- `API_BASE_URL`：サーバー専用（RSC / Route Handler）

テンプレのデフォルト：
- `.env.example` を用意しているため、プロジェクトでは `.env.local` を作って値を確定させる

ハードコード禁止：ホスト名/ポート番号は docs と env に集約し、ソース内に散らさない。

## ローカル起動手順（テンプレ）
```bash
# backend
docker compose up -d

# frontend
npm install
npm run dev
```

補足：フロントも Docker 化する場合は、
- `NEXT_PUBLIC_API_BASE_URL` を `http://host.docker.internal:xxxx` に寄せる（macOS）
- もしくは compose network 内でサービス名解決する（`http://gateway:8080` 等）

## CORS / Cookie / Proxy の方針
- 開発プロキシ：原則 **Next.js をBFFにして同一オリジン化** する（CORSを避ける）
	- 例：Next Route Handler で Go Gateway に中継
	- 例：Next rewrites で `/api/*` を upstream に転送
- Cookie 認証：
	- local は `SameSite=Lax` を基本（要件に応じて）
	- `Secure` は https 前提（local で https を使うかどうかを決めて明記する）

注意：Cookie 認証を「別オリジン」でやると、SameSite/ドメイン/CORS の組み合わせでハマりやすい。

## 禁止（AI向け）
- ドキュメントに書いていないポート番号・ホスト名を決め打ちしない
- API 仕様を推測してエンドポイントを手書きしない（OpenAPI -> 生成を優先）

## プロジェクト固有で確定させるチェックリスト
- compose のサービス名と exposed port
- local/staging の baseURL
- 認証方式（Cookie/Bearer）とトークン保管場所
- Next 側での proxy 方針（Route Handler / rewrites / なし）
