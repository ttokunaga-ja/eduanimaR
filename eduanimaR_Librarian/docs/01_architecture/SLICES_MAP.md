# SLICES_MAP

## 位置づけ
フロントエンド（Next.js + FSD）の **機能スライス (slices)** と、バックエンド（Go）の **マイクロサービス境界** の対応関係を定義する。

## FSD における「スライス」とは
- FSD の各階層（entities / features / widgets / pages）は、さらに**機能単位のディレクトリ（スライス）**に分割される
- 例:
  - `entities/user/` ← これが1つのスライス
  - `features/auth/login-form/` ← これも1つのスライス

## バックエンドとの対応
| FE Slice | BE Microservice | 説明 |
| --- | --- | --- |
| `entities/user/` | User Service | ユーザー情報の取得・表示 |
| `entities/product/` | Product Service | 商品情報の取得・表示 |
| `entities/order/` | Order Service | 注文情報の取得・表示 |
| `features/auth/login-form/` | User Service (Auth API) | ログインフォーム |
| `features/cart/add-to-cart/` | Order Service | カート追加 |
| `features/search/search-bar/` | Search Service (Elasticsearch) | 検索バー |

> 注: 上表は例。実プロジェクトの実態に合わせて必ず更新すること。

## 最小ルール
- 新機能追加時は、まず「どのバックエンドサービスから取得するか」を `MICROSERVICES_MAP.md` で判断する
- その後、フロントエンド側の「どの階層・スライスに置くべきか」を決める
  - データ取得・表示だけなら `entities`
  - ユーザー操作（フォーム送信等）があれば `features`
  - 複数の機能を組み合わせた大きなUIブロックなら `widgets`
- それでも不明な場合は、ドメイン境界（責務）の再定義を検討する

