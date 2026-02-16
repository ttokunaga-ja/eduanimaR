---
Title: AUTH_SESSION
Description: eduanimaRの認証・セッション管理方針（Phase別実装）
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-15
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
