---
Title: AUTH_SESSION
Description: eduanimaRの認証・セッション管理方針（Phase別実装）
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, authentication, session, sso
---

# AUTH_SESSION

## 目的
eduanimaRの認証・セッション管理方針を定義する。

## Phase別の認証方針

### Phase 1: ローカル開発（認証スキップ）
- **実装**: 固定の`dev-user`を使用
- **環境変数**: `NEXT_PUBLIC_AUTH_MODE=local`
- **実装場所**: `src/shared/api/auth-provider.ts`
- **注意**: 本番環境では絶対に使用しない

```typescript
// Phase 1例
if (process.env.NEXT_PUBLIC_AUTH_MODE === 'local') {
  return { userId: 'dev-user', authenticated: true };
}
```

### Phase 2以降: SSO認証

## Phase 2: SSO認証と拡張機能への誘導

### 新規ユーザー登録の境界
- **拡張機能**: 新規登録を許可（SSO認証 → Professor API `POST /auth/register`）
- **Web版**: 新規登録を禁止（SSO認証後、未登録ユーザーは拡張機能へ誘導）

### Web版での未登録ユーザー対応フロー
1. **SSO認証成功**（Google/Meta/Microsoft/LINE）
2. **Professor API呼び出し**: `POST /auth/login`
3. **Professor応答**: `AUTH_USER_NOT_REGISTERED`（ユーザーが未登録）
4. **フロントエンド処理**:
   - `/auth/register-redirect` へルーティング
   - 拡張機能誘導画面を表示
   - 誘導先URLを優先順位順に表示（Chrome Web Store → GitHub → 導入ガイド）

### 拡張機能誘導画面の実装（MUST）
実装例（実際のコードは `src/features/auth/ui/ExtensionInstallPrompt.tsx` に配置）:
```typescript
// 実装イメージ（ドキュメント用参考コード）
export const ExtensionInstallPrompt = () => {
  const extensionUrls = useExtensionUrls(); // shared/config/extension-urls

  return (
    <Box>
      <Typography variant="h5">
        eduanimaRをご利用いただくには、Chrome拡張機能のインストールが必要です
      </Typography>
      <Typography variant="body1">
        Web版は既存ユーザーのログイン専用です。新規登録は拡張機能から行ってください。
      </Typography>
      <Button 
        variant="contained" 
        href={extensionUrls.chromeWebStore}
        target="_blank"
      >
        拡張機能をインストール
      </Button>
      <Link href={extensionUrls.githubReleases} target="_blank">
        GitHubからダウンロード
      </Link>
      <Link href={extensionUrls.officialGuide} target="_blank">
        導入ガイドを見る
      </Link>
    </Box>
  );
};
```

### エラーコード定義（ERROR_CODES.md に追加）
| コード | 意味 | UI挙動 |
|--------|------|--------|
| `AUTH_USER_NOT_REGISTERED` | SSO認証成功だが未登録 | 拡張機能誘導画面へ遷移 |
| `AUTH_EXTENSION_REQUIRED` | Web版での新規登録試行を検知 | 同上 |

### Professor API契約（Phase 2で実装）
- **エンドポイント**: `POST /auth/login`
- **未登録ユーザーの応答**:
  - ステータスコード: `403 Forbidden`
  - レスポンス: エラーコード + 拡張機能誘導URL

#### 対応プロバイダー
- Google
- Meta（Facebook）
- Microsoft
- LINE

#### 実装方針
- **ライブラリ**: NextAuth.js（Auth.js v5推奨）
- **認証フロー**:
  1. ユーザーがログインボタンクリック
  2. OAuthプロバイダーへリダイレクト
  3. 認証完了後、Professor APIへトークン送信
  4. Professor側でトークン検証・ユーザー登録/ログイン
  5. フロントエンドにセッション返却

#### Professor API連携
- **エンドポイント**: `POST /v1/auth/verify`
- **リクエスト**: `{ provider: 'google', idToken: '...' }`
- **レスポンス**: `{ userId: 'uuid', accessToken: '...', refreshToken: '...' }`

#### トークン管理
- **保存場所**: HttpOnly Cookie（セキュア）
- **有効期限**: アクセストークン 1時間、リフレッシュトークン 7日
- **更新**: アクセストークン期限切れ時、リフレッシュトークンで自動更新

#### Chrome拡張とWebアプリの認証境界統一
- **共通セッションストア**: Cloud Firestoreまたは Professor DB
- **実装**: 拡張とWebアプリが同一のセッションIDを参照
- **ログアウト**: どちらかでログアウトすると両方無効化

## セキュリティ要件
- **トークン暗号化**: HTTPS通信必須
- **CSRF対策**: SameSite Cookie属性
- **XSS対策**: HttpOnly Cookie、CSP適用

## 実装チェックリスト
- [ ] Phase 1: 固定dev-user実装
- [ ] Phase 2: NextAuth.js設定
- [ ] Phase 2: Professor API `/auth/verify` 連携
- [ ] Phase 2: トークンリフレッシュ実装
- [ ] Phase 2: ログアウト処理
- [ ] Phase 2: Chrome拡張とWebアプリの認証統一
