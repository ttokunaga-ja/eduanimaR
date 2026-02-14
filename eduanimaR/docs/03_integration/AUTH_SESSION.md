# Auth & Session Policy（SSO / Cookie / CSRF / Cache）

このドキュメントは、Next.js（App Router）を BFF として運用する際の
OAuth 2.0 / OpenID Connect による SSO 認証、セッション管理、Cookie とキャッシュの相互作用を "契約" として固定します。

関連：
- DAL：`../01_architecture/DATA_ACCESS_LAYER.md`
- キャッシュ：`../01_architecture/CACHING_STRATEGY.md`
- セキュリティヘッダー/CSP：`SECURITY_CSP.md`
- Chrome拡張：`CHROME_EXTENSION_BACKEND_INTEGRATION.md`
- データモデル：`../01_architecture/DATA_MODELS.md`

---

## 結論（Must）

- 認証は **OAuth 2.0 / OpenID Connect** による SSO（対応プロバイダ: Google / Meta / Microsoft / LINE）
- セッションは **HttpOnly Cookie** を基本（アクセストークンを LocalStorage に置かない）
- **個人情報の最小化**：フロントエンドは `user_id` (UUID + nanoid) のみ保持
  - **重要**: メールアドレス、電話番号、表示名等の個人情報はProfessor側でも管理しない
  - Backend DBには `provider` と `provider_user_id` のみが保存される
- CSRF 方針（SameSite / Origin check / token）をプロジェクトとして固定する
- ユーザー依存データは "意図せず共有キャッシュ" されないようにする（動的化 or no-store か設計で担保）
- 認証状態の変更（login/logout）は UI に即反映されるよう、再検証と境界を明示する
- Chrome拡張とのセッション共有：Cookie認証による統合パターン
- TanStack Query と統合し、ログイン/ログアウトで適切にキャッシュを無効化する

---

## 1) SSO 認証（OAuth 2.0 / OpenID Connect）

### 1.1 対応プロバイダ

Phase 1 では以下のプロバイダをサポート：
- **Google**（推奨）
- **Meta（Facebook）**
- **Microsoft**（Azure AD / Personal accounts）
- **LINE**

### 1.2 認証フロー

1. **ログインボタン押下**
   - ユーザーがプロバイダを選択
   - 認証サーバーにリダイレクト（Authorization Code Flow）

2. **認証サーバーでの認証**
   - ユーザーが認証情報を入力（プロバイダのUIで実施）
   - スコープ同意画面を表示

3. **コールバック処理**
   - `/api/auth/callback/[provider]` で Authorization Code を受け取る
   - Professor API にコードを転送し、ユーザー情報を取得
   - **個人情報の最小化**: `provider` と `provider_user_id` のみをDBに保存
   - `user_id` (UUID) と `nanoid` (20文字) を生成

4. **セッション確立**
   - HttpOnly Cookie にセッション ID を保存
   - フロントエンドは `/app` へリダイレクト

### 1.3 個人情報の取り扱い

**重要な設計方針**：
- **Backend DBには個人情報を一切保存しない**（メール、電話番号、表示名等）
- Backend DBに保存されるのは `provider` と `provider_user_id` のみ
- ユーザー識別は `user_id` (UUID) で行う
- 外部公開には `nanoid` (20文字) を使用
- 認証プロバイダから取得した個人情報は、認証処理の完了後に破棄する

型定義（契約）：
```typescript
// shared/types/auth.ts (Backend DB Schemaと一致)
export interface CurrentUser {
  id: string;                // UUID (内部ID)
  nanoid: string;            // 20文字の外部公開ID
  provider: string;          // OAuth/OIDCプロバイダ（例: "google", "microsoft"）
  provider_user_id: string;  // プロバイダ側のユーザーID
  role: 'student' | 'instructor' | 'admin';
}
```

**UI表示について**：
- ユーザー名表示が必要な場合は、フロントエンド側で別途管理するか、省略する
- 例: "ユーザー（Google）" や "アカウント設定" のような表示

### 1.4 ログアウトフロー

1. **ログアウトボタン押下**
   - Server Action または Route Handler を呼び出し

2. **セッション破棄**
   - Cookie を削除
   - TanStack Query のキャッシュを全クリア（`queryClient.clear()`）

3. **リダイレクト**
   - ログインページまたは公開トップページへ遷移

---

## 2) Cookie ポリシー（テンプレ）

- `HttpOnly`: true（JS から読めない）
- `Secure`: true（本番）
- `SameSite`: `Lax`（基本）
- `Path`: `/`
- `Domain`: 必要時のみ
- `Max-Age`: セッションの有効期限（例: 7日間）

注意：Cookie 設計は CSRF/CORS/サブドメイン構成に依存するため、プロジェクト固有で確定させる。

---

## 3) CSRF（Must）

- Server Actions は UI 起点の mutation で第一選択（origin チェック等の枠組みがある）
- Route Handler を公開 API として使う場合：
  - CORS を明示（許可 origin を最小化）
  - Cookie 認証の場合は CSRF 対策を必ず入れる（token / double submit 等、方針を決める）

---

## 4) Cache との相互作用（Must）

- `cookies()` / `headers()` などの Dynamic API はルートを動的化し、Full Route Cache に影響する
- セッション Cookie の set/delete は "認証状態が変わる" ため、UI の整合性を崩しやすい
- ユーザー依存データの fetch では、キャッシュ設計を docs に残す（暗黙禁止）

運用上の目安：
- 認証必須ページ：動的化（意図して）
- 公開ページ：静的 + ISR / tag revalidate

---

## 5) TanStack Query との統合

### 5.1 ユーザー情報の取得

```typescript
// shared/api/auth.ts
export const authQueries = {
  currentUser: () => ({
    queryKey: ['auth', 'currentUser'] as const,
    queryFn: async (): Promise<CurrentUser | null> => {
      const res = await fetch('/api/auth/me');
      if (res.status === 401) return null;
      if (!res.ok) throw new Error('Failed to fetch user');
      return res.json();
    },
    staleTime: 5 * 60 * 1000, // 5分
    retry: false, // 401は再試行しない
  }),
};
```

### 5.2 ログイン/ログアウト時の再検証

```typescript
// features/auth/lib/useLogout.ts
export function useLogout() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async () => {
      await fetch('/api/auth/logout', { method: 'POST' });
    },
    onSuccess: () => {
      // すべてのキャッシュをクリア
      queryClient.clear();
      // ログインページへリダイレクト
      window.location.href = '/login';
    },
  });
}
```

---

## 6) Chrome拡張との認証共有

### 6.1 統合パターン

Chrome拡張（Sidepanel / Content Script）は、Web版と同じCookie認証を利用する。

- Web版でログイン → Cookie がブラウザに保存される
- 拡張は同じドメインのCookieを利用してAPIリクエストを送信
- 拡張側で独自の認証フローは不要（Web版でログイン済みであれば動作）

### 6.2 実装方針

拡張の Background Service Worker から Professor API を呼び出す場合：
- `credentials: 'include'` を指定してCookieを送信
- 同一オリジンであることを確保（CORS設定）

```typescript
// 拡張の Background Service Worker
const res = await fetch('https://example.com/api/v1/subjects', {
  credentials: 'include', // Cookie を送信
  headers: {
    'Content-Type': 'application/json',
  },
});
```

### 6.3 セッション切れの検知

拡張側で 401 エラーを受け取った場合：
- ユーザーに「Web版でログインしてください」と表示
- Web版のタブを開く導線を提供

関連：`CHROME_EXTENSION_BACKEND_INTEGRATION.md`

---

## 7) 実装責務の分離（Must）

- 認可（Authorization）は DAL に閉じ込める
- UI は認証/認可の "分類結果" を受けて分岐する（code で）

---

## 禁止（AI/人間共通）

- トークンを LocalStorage に保存
- ユーザー依存レスポンスを "共通キャッシュ" に載せる
- 例外を握りつぶして「ログインし直して」で済ませる（分類して扱う）
- **個人情報（メールアドレス、電話番号等）をフロントエンドまたはProfessorに保存する**
- OAuth プロバイダから取得した個人情報を認証後も保持する

---

## 実装チェックリスト

- [ ] SSO プロバイダ（Google/Meta/Microsoft/LINE）の設定が完了しているか？
- [ ] Cookie は HttpOnly / Secure / SameSite=Lax で設定されているか？
- [ ] フロントエンドで扱うユーザー情報は `provider` と `provider_user_id` のみか（個人情報なし）？
- [ ] メールアドレス等の個人情報が不要に保存されていないか？
- [ ] ログアウト時に TanStack Query のキャッシュがクリアされるか？
- [ ] Chrome拡張が Web版の Cookie を利用して認証できるか？
- [ ] 401エラー時にログインページへ適切に誘導されるか？
- [ ] CSRF 対策が実装されているか？
