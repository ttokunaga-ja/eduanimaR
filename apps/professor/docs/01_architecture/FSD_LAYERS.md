# FSD_LAYERS

## 位置づけ
フロントエンド（Next.js + TypeScript）で採用する **FSD (Feature-Sliced Design)** の階層ルールと、バックエンド（Go）との対応関係を定義する。

## FSD とは
- **Feature-Sliced Design**: 機能単位で縦に切り、階層単位で横に切る、2軸のディレクトリ設計手法
- 大規模開発でも **責務の混在を防ぎ、変更影響を局所化** できる
- `eslint-plugin-boundaries` で階層違反を自動検知する

## FSD の階層（下から上へ）
| 階層 | 役割 | 依存ルール | 配置例 |
| --- | --- | --- | --- |
| **shared** | 共通部品・設定 | どこからでも使える | `src/shared/api/`, `src/shared/ui/`, `src/shared/config/` |
| **entities** | ビジネス実体 | shared のみに依存 | `src/entities/user/`, `src/entities/product/` |
| **features** | 機能・ユースケース | shared + entities に依存 | `src/features/auth/login-form/`, `src/features/cart/add-to-cart/` |
| **widgets** | 大きなUIブロック | shared + entities + features に依存 | `src/widgets/header/`, `src/widgets/footer/` |
| **pages** | ページ単位の組み立て | 下位すべてに依存可 | `src/pages/user-profile/`, `src/pages/product-list/` |
| **app** | ルーティング・Providers | 原則 pages をimportするだけ | `src/app/` (Next.js App Router) |

## 重要な制約（MUST）
- **上位層から下位層への依存は禁止**（例: `entities` から `features` を import してはいけない）
- **同じ階層内の横断的 import も原則禁止**（例: `features/auth` から `features/cart` を直接 import しない）
- 違反は `eslint-plugin-boundaries` で CI で検知する

## バックエンドとの対比
| FE (FSD) | BE (Clean Architecture) |
| --- | --- |
| `app` (routing) | `cmd/` (entry point) |
| `pages` (page composition) | `transport/http` (OpenAPI + SSE) |
| `features` (use cases) | `usecase` (business logic) |
| `entities` (business entities) | `domain` (entities) |
| `shared/api` (generated) | `repository` interface |
| `shared/ui` (MUI components) | （該当なし） |

## eduanima+R 固有の注意（MUST）
- Frontend が直接呼ぶバックエンドは Professor のみ（OpenAPI + SSE）
- 生成回答は必ず Source を表示する（クリック可能な path/url + ページ番号等）

## ディレクトリ例
```
src/
├── app/                  # Next.js App Router
├── pages/                # FSD: Pages Layer
│   └── user-profile/
├── widgets/              # FSD: Widgets Layer
│   └── header/
├── features/             # FSD: Features Layer
│   └── auth/login-form/
├── entities/             # FSD: Entities Layer
│   └── user/
└── shared/               # FSD: Shared Layer
    ├── api/              # Orval生成コード
    ├── ui/               # MUIラッパー
    └── config/
```

