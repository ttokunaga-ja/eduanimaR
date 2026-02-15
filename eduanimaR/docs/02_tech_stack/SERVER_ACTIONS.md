# Server Actions & Mutations Policy（Next.js App Router / 2026）

このドキュメントは、Next.js（App Router）における mutation（更新系処理）を **Server Actions / Route Handlers のどちらで実装するか**を一意に決め、
- 余計な API（グルーコード）乱立
- キャッシュ再検証の破綻
- フォーム/エラーUXの不統一
- CSRF/CORS 等の取りこぼし
を防ぐための契約です。

関連：
- キャッシュ運用：`../01_architecture/CACHING_STRATEGY.md`
- 失敗の標準：`../03_integration/ERROR_HANDLING.md`
- エラーコード：`../03_integration/ERROR_CODES.md`
- DAL：`../01_architecture/DATA_ACCESS_LAYER.md`

---

## エグゼクティブサマリー（BLUF）

**「UIからの直接呼び出し（内部利用）」なら Server Actions**、**「外部システムからの呼び出し（公開API）」なら Route Handler**。

迷った場合は **Server Actions を第一選択**とします。
- フォーム（progressive enhancement）
- キャッシュ再検証（`revalidateTag` / `revalidatePath`）
- `redirect` / cookie 更新
と統合されており、運用の事故率が下がります。

---

## 結論（Must）

- UI（画面）起点の mutation は **原則 Server Actions**
- 外部（Webhook / Mobile / 3rd party）起点の mutation は **Route Handler**
- mutation の境界で **必ず再検証**する（`revalidateTag` / `revalidatePath`）
- 入力は **サーバ側で再検証**する（Zod 等）。クライアント入力を信用しない
- “期待される失敗（validation 等）” は **throw ではなく戻り値**で扱う（Next.js 推奨の expected error 方式）

---

## 1) 選択のための決定マトリクス

| 比較項目 | Server Actions | Route Handlers |
| :--- | :--- | :--- |
| 主なユースケース | アプリ内のフォーム送信、ボタン操作 | 外部API（Webhook、Mobile App）、REST API |
| 呼び出し元 | 自社 Next.js アプリの UI（Client/Server Components） | 外部システム、`curl`、サードパーティサービス |
| プロトコル | RPC（関数呼び出しのように振る舞う） | REST / HTTP標準 |
| 型安全性 | 引数/戻り値が共有されやすい | 手動（OpenAPI で契約化するのが基本） |
| フォーム連携 | 強力（`useActionState`、progressive enhancement） | 手動（`fetch` + 状態管理が必要） |
| HTTP 制御 | 制限あり（cookie/redirect は強い） | 完全制御（Status/CORS/Headers/Content-Type） |
| セキュリティ | origin チェック等の枠組みがある | CSRF/CORS 等を手動で設計 |

---

## 2) Route Handler を選ぶべき例外（Must）

- Webhook の受け口（Stripe/SendGrid 等）
- モバイルアプリ等のバックエンドとして同一 API を提供する
- 複雑な HTTP レスポンスが必要（特定 status、バイナリ、特殊 Content-Type、CORS 制御など）

---

## 3) Server Actions 実装ルール（Must）

### 3.1 配置と import
- Server Actions は **`"use server"` ファイル**に閉じ込める（Client Component 内に定義しない）
- actions は slice の責務に沿って配置する（例：`features/*/model/actions.ts` など）

### 3.2 入力検証
- `FormData` / 引数は **サーバ側で Zod で parse** する
- 期待される失敗は “エラー型を返す” ことで UI が分岐できるようにする（throw しない）

### 3.3 再検証（キャッシュ整合）
- 更新後は **必ず** `revalidateTag` か `revalidatePath` を呼ぶ
- `router.refresh()` と混同しない（`router.refresh` は Router Cache のみで、Data/Full Route を消さない）

---

## 4) expected error（戻り値）/ unexpected error（throw）

- expected error：validation 失敗、業務ルール違反（「起きうる」）
  - **戻り値**で UI に返す（例：`{ ok: false, fieldErrors, formError }`）
  - Client では `useActionState` で表示
- unexpected error：バグ、想定外例外（「起きないはず」）
  - throw して route の `error.tsx` に寄せ、運用に載せる

---

## 5) 最小テンプレ（Server Actions）

### 5.1 Action（server）

```ts
'use server'

import { revalidatePath, revalidateTag } from 'next/cache'
import { z } from 'zod'

const schema = z.object({
  name: z.string().min(1),
})

type ActionState =
  | { ok: true }
  | { ok: false; message: string; fieldErrors?: Record<string, string> }

export async function updateUsername(prevState: ActionState, formData: FormData): Promise<ActionState> {
  const parsed = schema.safeParse({ name: formData.get('name') })
  if (!parsed.success) {
    return { ok: false, message: '入力が不正です' }
  }

  // TODO: DAL 経由で更新 + 認可
  // await updateUserName(parsed.data)

  revalidatePath('/profile')
  // or: revalidateTag('user:current')

  return { ok: true }
}
```

### 5.2 Form（client）

```tsx
'use client'

import { useActionState } from 'react'
import { updateUsername } from './actions'

const initialState = { ok: true as const }

export function ProfileForm() {
  const [state, formAction, pending] = useActionState(updateUsername, initialState)

  return (
    <form action={formAction}>
      <input name="name" />
      {!state.ok && <p aria-live="polite">{state.message}</p>}
      <button type="submit" disabled={pending}>Save</button>
    </form>
  )
}
```

---

## 禁止（AI/人間共通）

- UI からの mutation を “とりあえず” Route Handler にする（グルーコード化）
- クライアント側だけで入力検証してサーバで検証しない
- mutation 後の再検証を忘れる（古い UI が残る）
- 期待される失敗を throw で処理する（error boundary に流す）
