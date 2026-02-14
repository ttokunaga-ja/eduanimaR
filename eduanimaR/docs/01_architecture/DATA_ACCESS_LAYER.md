# Data Access Layer（DAL）ポリシー

このドキュメントは、Next.js App Router（RSC）時代の「データ取得の置き場所」を固定し、
- 認可漏れ
- 秘匿情報の露出
- “どこでも何でも取得する” ことによる複雑性
を防ぐための **運用契約** です。

本テンプレートの前提スタックは [STACK.md](../02_tech_stack/STACK.md) を参照。

---

## 結論（Must）

- **Server Component（RSC）でのデータ取得は、原則として DAL 経由で行う**
- **DAL は server-only**（クライアントに import されるとビルドが落ちる状態を目指す）
- **DAL は認可チェックと DTO 最小化を責務に含める**
- **`process.env` / secret の参照は DAL に集約**（他レイヤーに散らさない）

---

## なぜ DAL が必要か（2026 / RSC 前提）

RSC により「サーバで自由に取得できる」ようになった反面、以下の事故が起きやすくなります。

- **“つい” Server Component から DB/API を直叩きしてしまい、認可が抜ける**
- Server → Client への props で **過剰なデータ（秘匿・個人情報）を渡してしまう**
- 取得箇所が散って **キャッシュ/再検証の設計が破綻** する

DAL を 1 箇所に寄せることで、監査・レビュー・変更が容易になります。

---

## 置き場所（推奨）

本テンプレートでは、API クライアント生成（Orval）を `src/shared/api/generated` に置く前提のため、
DAL は「生成物の上に薄い手書きレイヤー」として分離します。

推奨パターン（例）：

```text
src/shared/api/
├── generated/                  # 自動生成（手編集禁止）
├── client.ts                   # 共通設定（baseURL/認証/共通fetcher）
├── errors.ts                   # エラー分類
├── dal/                        # DAL（server-only）
│   ├── user.ts                 # getCurrentUserDTO 等
│   └── product.ts
└── index.ts                    # Public API
```

DAL の modules は先頭に `import 'server-only'` を置く運用を推奨します（依存して良いのは server 側のみ）。

---

## DAL の責務（Must）

### 1) 認可（Authorization）
- **毎回** “現在のユーザーがそのデータにアクセスしてよいか” をチェックする
- クライアントから来る入力（params/searchParams/formData など）は **信用しない**（必ず再検証）

### 2) DTO 最小化（Data Minimization）
- Client Component に渡すデータは **表示に必要な最小フィールドのみ**
- “バックエンドのレスポンスをそのまま props に渡す” を禁止

### 3) キャッシュ契約（CACHING_STRATEGY.md と整合）
- DAL は `fetch` のキャッシュ方針（`next.revalidate` / `next.tags` 等）を統一する起点
- invalidate は Server Action/Route Handler 側で明示する

関連：キャッシュの運用契約は [CACHING_STRATEGY.md](./CACHING_STRATEGY.md)

---

## 禁止（AI/人間共通）

- RSC が Route Handler を呼ぶ（サーバ内で **余計な HTTP hop** を作る）
- Server Component から取得した “生データ” を Client に丸渡しする
- secrets / `process.env` を DAL 以外で参照する（`NEXT_PUBLIC_` 以外）

---

## 監査チェック（最低限）

- `"use client"` ファイルが server-only module を import していないか
- Client props の型が過剰に広くないか（バックエンドの型をそのまま使っていないか）
- DAL が “認可” と “DTO 最小化” を必ずしているか
