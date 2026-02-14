# Caching & Revalidation Strategy（Next.js App Router）

このドキュメントは、Next.js（App Router）の複数キャッシュ（Data/Route/Router）を前提に、
「どこで」「何を」「どの粒度で」キャッシュし、いつ無効化するかを固定するための契約です。

関連：データ取得の置き場所は [DATA_ACCESS_LAYER.md](./DATA_ACCESS_LAYER.md)

---

## 結論（Must）

- 取得は **Server Component（RSC）優先 + DAL 経由** を基本とする
- “更新” があるデータは **tag で無効化（revalidateTag）できる設計** を優先する
- “ユーザー依存” のデータは、意図せず静的化しない（Dynamic API で動的化するか、no-store を明示）
- キャッシュ方針（静的/動的/再検証）は **ドキュメントに残す**（暗黙禁止）

---

## Next.js のキャッシュ（概念整理）

App Router では、用途の異なるキャッシュが複数存在します。

- **Data Cache**：`fetch` の結果をキャッシュ（`next.revalidate` / `next.tags` 等で制御）
- **Full Route Cache**：ルート全体のレンダー結果をキャッシュ（静的レンダリング時に効く）
- **Router Cache**：クライアント遷移時のルートセグメントのキャッシュ（体感速度に影響）

重要：`cookies()` や `headers()`、`searchParams` 等の Dynamic API の使い方次第で、
ルート全体が動的化（＝静的キャッシュされない）します。

---

## 追加：Next.js の公式ポイント（2026 / Must）

### Request Memoization（同一 request 内の重複排除）

- Server では同一 URL + options の `fetch(GET/HEAD)` は、React により **同一リクエスト内で自動重複排除**される
- 「Layout で取って props でバケツリレー」を必須にしない（必要な箇所で取得してもよい）

### `react/cache` の使い所

- `fetch(GET/HEAD)` は自動 memoize されるため、基本は `react/cache` は不要
- DB クライアント等、memoize されない “関数” を同一 request 内で重複排除したい場合に使う

### `revalidatePath` vs `router.refresh`

- `revalidatePath` / `revalidateTag`：**Data Cache / Full Route Cache を purge** し、次回から確実に新しいデータへ
- `router.refresh()`：**Router Cache を無効化して再描画**するが、Data Cache / Full Route Cache は消さない

---

## 追加：Dynamic API（動的化）のチェック表（Must）

意図せず Full Route Cache を無効化しないため、以下をレビューで確認する。

- `cookies()` / `headers()` を root layout で使っていないか
- `searchParams` 依存の処理が “本当に必要なセグメント” に閉じているか
- 認証必須ページが静的化されていないか（逆に公開ページを動的化していないか）

---

## 方針（Must / Should）

### A. 公開ページ（SEO/集客に効く）
- **基本：SSR + 適切な再検証（revalidate）**
- 更新頻度に応じて `revalidate` を決める
- 更新のトリガが明確な場合は `next.tags` を付与し、更新後に `revalidateTag` で確実に更新する

### B. 認証必須・ユーザー依存ページ
- **基本：動的レンダリング（意図した上で）**
- キャッシュを使う場合も “ユーザー単位に分離される” ことを前提にする
- 誤って全ユーザー共通キャッシュにならないように、取得と key/tag の設計を明文化する

### C. 変更がない静的コンテンツ
- **SSG/静的化** を優先（コスト/速度が良い）
- ISR を使う場合は「再生成遅延・古いデータ許容」を合意しておく

---

## tag 命名規約（推奨）

tag は “人間が意味を読める” ことを優先します。

例：
- `product:{productId}`
- `product:list`
- `user:{userId}`
- `cart:{userId}`

注意：tag を “画面名” に寄せない（再利用不能になる）。

---

## 無効化の責務（Must）

- revalidate は **mutation の境界（Server Action / Route Handler）** で行う
- UI（Client Component）側で「なんとなく更新」をやらない

更新後の整合は、
- Next のキャッシュ無効化（`revalidateTag` / `revalidatePath`）
- TanStack Query の invalidate（`invalidateQueries`）
を **二重管理しない** 方針をとる

原則：
- “RSC で取得するデータ” は Next の再検証で整合させる
- “Client hooks が主（常時インタラクティブ）” のデータは Query を主にする

---

## TanStack Query SSR/Hydration との整合

本テンプレートでは SSR/Hydration を **Must** とします。
ただし 2026 年時点では、RSC により「hydration する JS を最小化」するのが実務の最適解です。

- 初期表示で必要なデータは **Server で prefetch → dehydrate → HydrationBoundary** を基本にする
- ただし “RSC で完結できる表示” は、Client 化せず SSR のみで終える（hydration コストを払わない）

詳細：SSR/Hydration の概念と採用基準は [../02_tech_stack/SSR_HYDRATION.md](../02_tech_stack/SSR_HYDRATION.md)

---

## 禁止（AI/人間共通）

- `no-store` / `revalidate: 0` を理由なく乱用する（コスト増）
- Dynamic API を Root Layout で使って “全ルートを動的化” する（意図せず性能劣化）
- RSC が Route Handler を呼ぶ（サーバ内で無駄な hop）
