# COMPONENT_ARCHITECTURE

## 位置づけ
フロントエンド（Next.js + MUI v6 + Pigment CSS）のコンポーネント設計ルールを定義し、バックエンド（Go）との対応関係を明確にする。

## フロントエンドのコンポーネント責務
| 階層 | 責務 | 状態管理 | 例 |
| --- | --- | --- | --- |
| `shared/ui/` | 汎用UIコンポーネント（MUIラッパー） | 原則なし（props受け取りのみ） | `Button`, `Input`, `Card` |
| `entities/<entity>/ui/` | 実体の表示コンポーネント | TanStack Query でサーバー状態取得 | `UserCard`, `ProductCard` |
| `features/<feature>/ui/` | 機能のUIコンポーネント | React Hook Form + Zod でフォーム管理 | `LoginForm`, `AddToCartButton` |
| `widgets/<widget>/ui/` | 大きなUIブロック | 下位層の状態を組み合わせる | `Header`, `Footer` |
| `pages/<page>/ui/` | ページ全体 | RSC で初期データ取得 | `UserProfilePage` |

## MUI + Pigment CSS の制約（重要）
- **Pigment CSS**: ゼロランタイム CSS。ビルド時にスタイルを生成し、実行時の JS 負荷を減らす。
- **制約**:
  - 動的なスタイル変更（`sx` prop）の使用を `shared/ui/` 内のコンポーネントに限定する
  - 上位層（features/widgets/pages）では、可能な限り事前定義されたコンポーネントを組み合わせるだけにする
- **詳細**: 同階層の `MUI_PIGMENT.md` を参照（DO/DON'T、アップグレード時の確認観点）

## 状態管理
- **サーバー状態**: TanStack Query v5/v6（Orval生成のHooksを使用）
- **フォーム状態**: React Hook Form + Zod
- **グローバルUI状態**: Zustand（必要最小限。例: モーダル開閉、テーマ切り替え等）

## バックエンドとの対比
| FE | BE (Clean Architecture) |
| --- | --- |
| `pages/<page>/ui/` | `handler` (HTTP layer) |
| `features/<feature>/` | `usecase` (business logic) |
| `entities/<entity>/` | `domain` (entities) |
| `shared/api/` (Orval生成) | `repository` interface |
| `shared/ui/` (MUI) | （該当なし） |

## 実装例
```tsx
// src/entities/user/ui/UserCard.tsx
import { useGetUser } from '@/shared/api/user.gen'; // Orval生成
import { Card, Typography } from '@/shared/ui';

export const UserCard = ({ userId }: { userId: string }) => {
  const { data: user, isLoading } = useGetUser(userId);
  if (isLoading) return <div>Loading...</div>;
  return (
    <Card>
      <Typography variant="h5">{user?.name}</Typography>
    </Card>
  );
};
```

