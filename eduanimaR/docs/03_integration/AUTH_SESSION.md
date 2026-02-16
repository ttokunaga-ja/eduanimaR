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

### SSO連携方針（Handbookより）

eduanimaRは、以下のSSO方針に基づきます：

- **SSO-only認証**: Google/Meta/Microsoft/LINE（パスワードを保存しない）
- **データ最小化**: メールアドレス等を必須登録情報として求めない
- **セキュリティ・分析のみ**: ロケーション/IP/ログは、セキュリティ・分析のみに使用
- **コンテンツ取り扱い**: 抽出・派生データ（OCR、埋め込み、要約）は元データと同等に機密扱い
- **デフォルト個人利用**: すべてのコンテンツ・履歴はユーザー専用（明示的共有まで非公開）

**参照**: [`../../eduanimaRHandbook/01_philosophy/PRIVACY_POLICY.md`](../../eduanimaRHandbook/01_philosophy/PRIVACY_POLICY.md)

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

## 認証境界の保証（Must）

### Chrome拡張機能とWebアプリの認証統一

- **同一セッション**: Chrome拡張機能とWebアプリは同一のセッションを共有
- **同一user_id**: どちらの導線でも同一のuser_idで認可判定
- **認証状態の管理**: Professorが認証状態を管理(フロントエンドは認証トークンのみ保持)

### Phase 1の特殊対応

**ローカル開発時の認証スキップ**:
- 固定のdev-userを使用
- 認証UIは実装しない
- Professor側で自動的にdev-userセッションを生成

```typescript
// Phase 1での開発用設定例
if (process.env.NEXT_PUBLIC_AUTH_MODE === 'local') {
  // Professor側が自動的にdev-userを設定
  // フロントエンドは何もしない
}
```

**参照元SSOT**:
- `../../eduanimaRHandbook/01_philosophy/PRIVACY_POLICY.md` (SSO/OAuth)
- `../../eduanimaRHandbook/04_product/ROADMAP.md` (Phase 1-2の認証方針)

### Phase別の認証方針（Handbookより）

**Phase 1（ローカル開発）**:
- 認証スキップ（固定dev-user使用）
- 認証UIは実装しない

**Phase 2以降（本番環境）**:
- SSO認証実装（Google/Meta/Microsoft/LINE）
- Web版からの新規登録は禁止、拡張機能でのみユーザー登録可能
- **データ最小化**: メールアドレス等を必須登録情報として求めない
- **セッション管理**: Cookie（httpOnly, Secure, SameSite=Lax）

**参照**: [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md)、[`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)

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
